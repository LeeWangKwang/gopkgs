// vi: sw=4 ts=4:
/*
 ---------------------------------------------------------------------------
   Copyright (c) 2013-2015 AT&T Intellectual Property

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at:

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
 ---------------------------------------------------------------------------
*/


/*
------------------------------------------------------------------------------------------------
	Mnemonic:	ostack_hosts
	Abstract:	Functions to support getting physcial host and hypervisor maps and information.
					
	Date:		16 December 2013
	Author:		E. Scott Daniels
	Mods:		13 Aug 2014 - Changes to List_hosts() to allow multiple types to be supplied,
					and to work with the network centric list host function now that nova
					seems decoupled from the network information.
				04 Nov 2014 - Changes to better support grizzly.
				04 Dec 2014 - Now supports getting a list of physical hosts that are "enabled"
					and up (which we assume might not be down for maintenance).
				24 Jun 2014 - Combined List_hosts and List_enabled_hosts into a single function
					with small public facing wrappers.
				15 Jul 2015 - Corrected the reverse setting of the 'all' boolean in the list
					enabled hosts function call to list_hosts.
------------------------------------------------------------------------------------------------
*/

package ostack

import (
	"bytes"
	"fmt"
	"strings"
)

// ------------- structs that are used to unbundle the json response data -------------------

// this is based on data snarfed from doc: http://api.openstack.org/api-ref-compute.html
//

// ---------------- generated by os-hypervisors get ----------------------------
type ost_hypervisor struct {
	Hypervisor_hostname string
	Id	int
}

type ost_hypervisor_resp struct {
	Hypervisors []ost_hypervisor
}

// ------------------------------------------------------------------------------
type ost_hyp_service struct {
	Host string
	Id int
}

type ost_hyp_details struct {
	//cpu_info "?",
	//current_workload 0,
	//disk_available_least null,
	//free_disk_gb 1028,
	//free_ram_mb 7680,
	Hypervisor_hostname string
	Hypervisor_type string
	//hypervisor_version 1,
	//id 1,
	//local_gb 1028,
	//local_gb_used 0,
	//memory_mb 8192,
	//memory_mb_used 512,
	//running_vms 0,
	Service	ost_hyp_service
	//vcpus 1,
	//vcpus_used 0
}

type ost_hyp_details_resp struct {
	Hypervisors []ost_hyp_details
}

// -- internal helper stuff -----------------------------------------------------

/*
	Creates a map of various openstack service types that can be used to easily
	match a type name returned by openstack to a desired type bit mask supplied
	by the user. Seems that different strings come back depending on the request
	so this should provide a map from all possible names to the smaller set of
	constants this package provides to the user.
*/
func gen_svc_match_map( ) ( match_type map[string]int ) {

	match_type = make( map[string]int )

	match_type["compute"] = COMPUTE
	match_type["nova-compute"] = COMPUTE
	match_type["scheduler"] = SCHEDULE
	match_type["nova-scheduler"] = SCHEDULE
	match_type["network"] = NETWORK
	match_type["cert"] = CERT
	match_type["nova-cert"] = CERT
	match_type["cells"] = CELLS
	match_type["conductor"] = CONDUCTOR
	match_type["nova-conductor"] = CONDUCTOR
	match_type["consoleauth"] = AUTH
	match_type["nova-consoleauth"] = AUTH

	return
}


/*
	This is the real list_hosts/list_enabled_hosts work horse.
	Returns a list of hosts matching the htype which is an OR'd set of values.
	If all is true then all hosts encountered are listed, otherwise only the
	enabled/up hosts are listed.

	Lists only hosts associated with services which _might_ just differ from the hosts
	listed by os-hosts.  Certainly the information returned by  os-hosts has less
	information (running or not seems unimportant etc.), so this list might be more
	useful as it will return only those hosts that we _think_ are actually alive and
	well (which might not be true since the state seems based on the openstack software
	running on the physical host and not the state of the host itself).
*/
func (o *Ostack) list_hosts( htype int, all bool ) ( hlist *string, err error ) {
	var (
		resp_data	generic_response	// "root" of the response goo after pulling out of json format
		//jdata	[]byte					// raw json response data
		seen	map[string]bool			// used to weed duplicates
	)

	hlist = nil						// if we error, ensure nil list returned
	err = nil;
	s := ""							// tmp string to build list in
	sep := ""						// separater between list elements, nothing for first

	seen = make( map[string]bool )
	match_type := gen_svc_match_map()	// make matching matrix based on expected ostack strings

	err = o.Validate_auth()						// reauthorise if needed
	if err != nil {
		return
	}

	if o.chost == nil || *o.chost == "" {
		err = fmt.Errorf( "no chost url for ostack struct: %s", o.To_str( ) )
		return
	}


	if  htype & L3 != 0 {							// since networking is a separate request to neutron, make only if user set
		if o.nhost == nil || *o.nhost == "" {
			err = fmt.Errorf( "no nhost url for ostack struct: %s", o.To_str( ) )
			return
		}

		hlist, seen, err = o.List_l3_hosts( seen, true ) 	
		if *hlist == "" {								// no network hosts, can't be, so we assume it's pre neutron
			hlist, seen, err = o.List_l3_hosts( seen, false ) 	
		}
		if htype == L3 || err != nil  {			// when only network is requested, we can short out here.
			return
		}

		s = *hlist										// seed for the call to nova
		if len( seen ) > 0 {							// if something found in the list, sep is now space (bug fix 2014.08.30)
			sep = " "
		}
	}

	if  htype & NETWORK != 0 {							// since networking is a separate request to neutron, make only if user set
		if o.nhost == nil || *o.nhost == "" {
			err = fmt.Errorf( "no nhost url for ostack struct: %s", o.To_str( ) )
			return
		}

		hlist, seen, err = o.List_net_hosts( seen, true ) 	
		if *hlist == "" {								// no network hosts, can't be, so we assume it's pre neutron
			hlist, seen, err = o.List_net_hosts( seen, false ) 	
		}
		if htype == NETWORK || err != nil  {			// when only network is requested, we can short out here.
			return
		}

		s = *hlist										// seed for the call to nova
		if len( seen ) > 0 {							// if something found in the list, sep is now space (bug fix 2014.08.30)
			sep = " "
		}
	}

	body := bytes.NewBufferString( "" )

	url := fmt.Sprintf( "%s/os-services", *o.chost )		// tennant id is built into chost
	err = o.get_unpacked( url, body, &resp_data, "list_hosts:" )
	if err != nil {
		return
	}

	if resp_data.Error != nil {
		err = fmt.Errorf( "%s", resp_data.Error );
		return
	}

	if resp_data.Forbidden != nil {
		err = fmt.Errorf( "%s", resp_data.Forbidden );
		return
	}

	for k := range resp_data.Services {
		bin := strings.ToLower( resp_data.Services[k].Binary )
		if  match_type[bin] & htype != 0 {								// this type requested on call
			if !seen[resp_data.Services[k].Host] &&
				(all ||
				 (strings.ToLower( resp_data.Services[k].State ) == "up" &&
				  strings.ToLower( resp_data.Services[k].Status ) == "enabled")) {

				seen[resp_data.Services[k].Host] = true
				s += sep + resp_data.Services[k].Host
				sep = " "
			}
		}
	}

	hlist = &s
	return
}

// ---------------- public -------------------------------------------------------------------------------

/*
	Generates a pointer to a string containing a space separated list of physical host names
	that are associated with the type(s) passed in. Htype is one or more of the following
	types OR'd together if desired:
		L3, NETWORK, COMPUTE, SCHEDULE, AUTH, CONDUCTOR, CELLS, and CERT

	Duplicates host names, hosts that might have different functions, are removed from the
	list. The credentials associated with the object must have admin privlidges or odd results
	(an empty list or nil pointer) will result.
*/
func (o *Ostack) List_hosts( htype int ) ( hlist *string, err error ) {
	return o.list_hosts( htype, true )
}

/*
	Returns a space separated list of host names as a string. See List_Hosts for a description
	of values for htype. Only hosts which are indicated as both "up" and "enabled"
	are included in the list.
*/
func (o *Ostack) List_enabled_hosts( htype int ) ( hlist *string, err error ) {
	return o.list_hosts( htype, false )
}

/*
	Creates a map of hypervisor IDs to host names
*/
func (o *Ostack) Mk_hyp2host(  ) ( hmap map[int]*string, err error ) {
	var (
		hyp_data	ost_hypervisor_resp
	)

	hmap = nil
	err = nil;

	err = o.Validate_auth()						// reauthorise if needed
	if err != nil {
		return
	}

	//jdata = nil
	body := bytes.NewBufferString( "" )

	url := fmt.Sprintf( "%s/os-hypervisors", *o.chost )		// tennant id is built into chost
	err = o.get_unpacked( url, body, &hyp_data, "list_hosts:" )
	if err != nil {
		return
	}


	hmap = make( map[int]*string )
	for k := range hyp_data.Hypervisors {
		dup_str := hyp_data.Hypervisors[k].Hypervisor_hostname
		hmap[hyp_data.Hypervisors[k].Id] = &dup_str
	}

	return
}
