// vi: sw=4 ts=4:

/*
------------------------------------------------------------------------------------------------
	Mnemonic:	ostack.go
	Abstract: 	This module contains the generic functions used by the rest of the package.
				The other modules provide one request type
				and are broken out only to keep the number of structures (used to hack the 
				json into) defined in each file to a minumum.

	Date:		16 December 2013
	Authors:	E. Scott Daniels

	Mods:		23 Apr 2014 - Added tenant id support
				06 Jun 2014 - Removed cvt_dashes from Send_req as tagging can be used instead.
				28 Jul 2014 - Changed tenant_id to project ID. 
				11 Aug 2014 - Added stripping of v2.0 or v3 from end of host url. 
				19 Aug 2014 - Added scan of json for non-jsonish things. 
				28 Oct 2014 - Added support for identity requests as admin.
				04 Dec 2014 - To support generating a list of hosts that are 'active'.
				06 Jan 2015 - Additional nil pointer checks.
				03 Feb 2015 - Correct bad tag in structure def.
------------------------------------------------------------------------------------------------
*/

/*
	Interface to the openstack environment.
*/
package ostack

import (
	//"bufio"
	"bytes"
	//"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	//"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)


// ------------- structs that the json returned by openstack maps to ----------------------

/*
	Unfortunately the easiest way, from a coding perspective, to snarf and make use of the 
	json returned by OpenStacks's api is to define structs that match certain parts of the 
	json.  We only have to define fields/objects that we are insterested in, so additions
	to the json will likely not break things here, but if a field name that we are interested
	in changes we'll fall over. (I see this as a problem regardless of the language that 
	is used to implement this code and not a downfall of Go.) 
	
	The json parser will insert data only for fields that are externally visiable in these
	structs (capitalised first character).  The remainder of the field names match those 
	in the json data.  We can also insert non-exported fields which are unaffected by 
	the parser. 
	
	We hope this isn't true of openstack....
	Names that cannot be legally mapped to Go variable names (e.g.
	dst-port).  We can (blindly for now) convert all '-' characters followed by a lowercase
	alpha character to _. This will have the side effect of changing the actual data in 
	addition to the names, which will probably break things. 

	A side effect of using the in-built json functions of go is that all of the elements 
	of the structs must be externally accessable. 

	So.... each module in this package will end up needing (possibly) its own set of structs that
	are used to unpack json into and then have as a direct reference to the resulting output
	from openstack.  They are fragile in the sense that if ostack changes a field name
	we break. If fields are added, the result is that we ignore them which shouldn't break
	our function unless for some reason it is necessary to all of a sudden have that new
	field's data.

	These are pulled from doc at:
		http://developer.openstack.org/api-ref.html
*/


/*
	Both openstack and some proxy boxes return an error struct as a top level object
	in the json with the _same_ name (error), but with a differing representation of
	internal fields with duplicate names (code).  This is a more generic structure
	and thus the functions in error.go must be used to extract values with minimal
	effort (code() and String()). 
*/
type error_obj struct {			// generic error to snarf in http errors will be nil on success
	Message	string
	Code	interface{}
	Title	string
}

type proxy_error struct {		// allows us to attempt to unpack an error from the proxy
	Message string
	Code	string
}

/*
	Openstack structs used to unpack json it returns. These are mostly v2 interfaces. V3
	related structs all have an osv3_ prefix and are defined in a separate module
*/

// --- authorisation -----------

type ost_auth_endp struct {
	Tenantid	string;
	Publicurl	string;
	Internalurl	string;
	Adminurl	string;
	Region		string;
	Version		string;
	Versioninfo	string;
	Versionlist	string;	
}

type ost_auth_svccat struct {
	Name	string;
	Type	string;
	Endpoints	[]ost_auth_endp;
}

type ost_auth_user struct {
	Id		string;
	Name	string;	
	//Roles []ost_role		// unused by tegu so not defined
}

// ---  Mostly for ostack_tenant.go ---
type ost_tenant struct {
	Id 	string
	Name string
	Description string
	Enabled bool
}

type ost_meta struct {
	Is_admin int
}

type ost_token struct {
	Issued_at string
	Expires	string
	Id 		string
	Tenant	*ost_tenant
}

type ost_role struct {
	Name	string
	Id 		string
	Description string
    //"roles_links": []
}

type ost_user struct {
	Username string				// human readable name
	Roles	[]*ost_role
	Id		string;				// jibberish uuid
	Name	string;	
}

type ost_access struct {
	Token		*ost_token
	User		*ost_user
	Metadata	*ost_meta
	Servicecatalog	[]ost_auth_svccat;
}

// ---- ostack_hosts.go ----

// these are generated by the os-hosts request
type ost_os_host struct {
	Host_name	string
	Service		string
	Zone		string
}


// ---- ostack_net.go -----------------

// substruct to agent
type ost_net_config struct {
		Public 					*string 			// something like br-ex, so not sure what it is
		Devices 				int
		Vmprivate 				*string 
		Use_namespaces 			bool
		Gateway_external_network_id *string
		Handle_internal_only_routers bool
		Router_id 				*string
		Ex_gw_ports				int
		Interface_driver		*string
		Interfaces 				int
		Routers 				int
		Floating_ips 			int
		Dhcp_driver 			*string
		Dhcp_lease_time 		int
		Subnets					int
		Networks				int
}

// returned by v2.0/agents
type ost_net_agent struct {
       Started_at 		*string 
       Heartbeat_timestamp *string 
       Topic 			*string 
       Binary 			*string 
       Created_at		*string 
       Host				*string 
       Description		*string 
       Id 				*string 
       Configurations 	ost_net_config
       Agent_type 		*string 
       Alive 			bool 
       Admin_state_up 	bool 
}



// ---- network information ----------
type ost_network struct {
	Status 		string
	Subnets 	[]string
	Name 		string
	//admin_state_up
	Tenant_id	string
	Id			string
	//shared	bool
	Phys_net	string	`json:"Provider:physical_network"`	// tag to handle stupid json names given by ostack
	Phys_type	string	`json:"Provider:network_type"`		// vlan, vxlan, gre, etc
	Phys_seg_id	int		`json:"Provider:segmentation_id"`		// this will be vlan id
}


// ---- subnet information -----------
type ost_pool struct {
	End		string
	Start	string
}

type ost_subnet struct {
	Allocation_pools []ost_pool
	Cidr 			string
	Gateway_ip 		string
	//Host_routes 	[]string	// this is not an array of string that comes back from openstack -- array of objects
	Name 			string
	Id 				string		// this is the id listed in output from v2/networks in the subnet list
	Network_id 		string		// who knows what this ID really is
	Tenant_id 		string
}

type ost_subnet_list struct {
	Subnets 		[]ost_subnet
}


// -- structs for fetch interface stuff ------
type ost_fixed_ip struct {
	Subnet_id	string		// uuid
	Ip_address	string		// dot address
}

type ost_ifattach struct {
	Port_state	string		// "ACTIVE"
	Fixed_ips	[]*ost_fixed_ip
	Net_id		string		// uuid
	Port_id		string		// uuid
	Mac_addr	string		// colon sep mac address
}

type iface struct {
}

// --- port related things -----

type ost_os_port struct {
	Status			string
	Bind_host_id	string	`json:"binding:host_id"`			// assume this is the physical host name
	Bind_vif_type	string	`json:"binding:vif_type"`
	//Bind_capabilities	port_abilities	`json:"binding:capabilities"`
	Name			string
	//Admin_state_up bool
	Network_id		string
	Tenant_id		string
	//extra_dhcp_opts [] ???
	Device_owner	string
	Mac_address		string
	Fixed_ips		[]*ost_fixed_ip
	Id				string
	//Security_groups []string
	Device_id		string
}

// -- ostack_vms.go ---------

/*
	Openstack has a bug IMHO:  
		"addresses": {
				"xxxx": [ { addr/ip-pairs } ]
		}

		The problem is that xxxx is variable and should be static; The xml gets it right, and the json
		_should_ be something like:

		"addresses": {
			"network" : [
				{ 
					"id": "<network-name(xxxx)>",
					"addr":	"<ipaddress>",
					"version": <addr-version>
				}
			]
		}

	In order to dig out the IP address we have to jump through some hoops since we cannot change the 
	field name in a struct on the fly. Thus, we must declare the Address field as an interface and 
	parse out the data ignoring the variable name.  We blindly use the first network information that 
	is presented. 
*/

type ost_vm_addr_pair struct {
	Addr	string
	Version int
}
type ost_vm_addr struct {
	Network	[]ost_vm_addr_pair
}

type ost_server_meta struct {
 	Host_name	string	`json:"My Server Name"`			// bloody python programmers who use spaces in field names should be shot
}

type ost_addr struct {
	Addr	string
}

type ost_vm_server struct {			// we don't use all of the data; fields not included are commented out
	Azone		string	`json:"OS-EXT-AZ:availability_zone"`
	Accessipv4	string				// huh?
	Accessipv6	string
	Addresses	map[string][]ost_addr
	Created		string
	//flavour
	Hostid		string
	//image		<struct>
	//links		<array>			// these appear to be URLs to who knows what, not network oriented links
	//Metadata	ost_server_meta
	Name		string
	//properties <int>
	Status		string
	Tenant_id	string
	//updated	<string>
	//user_id	<string>
	Id			string

	// these are NOT documented on the openstack site -- sheesh
	Host_name	string	`json:"OS-EXT-SRV-ATTR:host"`		// who signed off on this field name?  hate openstack.
}


/*
	Service type retured by os-services api call. Meaning of fields is in the doc is in true
	openstack form: lacking. Most of our interpretations are guesses.
*/
type ost_service struct {
	Binary 	string				// service type e.g. "nova-scheduler"
	Host 	string				// host it's running on
	State 	string				// state and status wtf? "up" we assume "down" exists too
	Status 	string				// again??? "disabled", is enabled a possibility?
	//Updated_at "2012-10-29T13:42:02.000000",
	//Zone "internal"
}

/*
	External gateway info. (may have more fields, but this is what was documented)
*/
type ost_gwinfo struct {
	Network_id	string
}

/*
	Information returned by v2.0/routers
*/
type ost_router struct {
	Status 		string				// "ACTIVE", other values undocumented in os doc :(
	Name		string				// we assume the human readable name
	Admin_state_up bool
	Tenant_id 	string
	Id 			string
	External_gateway_info *ost_gwinfo	// unknown what this might be (no doc)
}

type ost_aggregate struct {
	Availability_zone 	string
	Created_at 	string
	Deleted 	bool 
	Deleted_at 	string
	Hosts 		[]string
	Id 			int
	//Metadata { "availability_zone "nova" },
	Name 		string
	Updated_at 	string
}

/*
	A collection of things that might come back from the various ostack calls. Should serve
	as a receiver for unbundling any json response.
*/
type generic_response struct {
	Access		*ost_access
	Aggregates	[]ost_aggregate
	Error		*error_obj
	Hosts		[]ost_os_host
    Interfaceattachments	 []ost_ifattach
	Networks 	[]ost_network
	Ports		[]ost_os_port
	Roles		[]ost_role
	Routers		[]ost_router				// from v2.0/routers
	Router		*ost_router					// from v2.0/routers/<routerid>/l3-agent
	Servers		[]ost_vm_server
	Services	[]ost_service				// list of services from os-service
	Subnets		[]ost_subnet
	Tenants		[]ost_tenant
	Agents		[]ost_net_agent
}


// -- our structs ----------------------------------------------------------------------------
type Ostack struct {
	token	*string			// authorisation token (could be very very large)
	small_tok	*string		// small token if the token is absurdly huge (it's the md5 of the huge one)
	expiry	int64			// timestamp when we assume the authorisation expires
	host	*string			// the general host (probabl only for auth queries) (should NOT include v2.0 or v3)
	chost	*string			// the url used to make compute oriented queries (returned by auth)
	nhost	*string			// the url used to make netork oriented api queries (returned by auth)
	ihost	*string			// url for the identity (keystone) service
	iahost	*string			// url for the identity (keystone) admin service
	passwd	*string
	user	*string
	project	*string			// project (tenant) name
	aregion	*string			// the authenticated region if a keystone is shared between sites
	project_id	*string
	user_id *string
	//tok_cache	map[string]*int64	// cache of user tokens that we've validated; reference is to an expiration time.
	tok_isadmin	map[string]bool		// maps token to whether or not it was identified as an admin
	isadmin	bool				// true if the authorised user associated with the struct is an admin
}

// ---- necessary globals --------------------------------------------------------------------
var (								// counters used by ostack_debug functions -- these apply to all objects!
	dbug_json_count int = 50		// set >= 10 to prevent json dumping of first 10 calls
	dbug_url_count int = 50			// set to 10 to prevent url dumping of first 10 calls
)

/*
	Build the main object which is then used to drive each type of request. 

	Region is the value used to suss out various endpoints. If nil is given, 
	then the user may call Authorise_region() with a specific region, or 
	use Authorise() to use the first in the list (default). If region is 
	provided here, then it is used on a plain Authorise() call, or when 
	the credentials are reauthenticated.
*/
func Mk_ostack_region( host *string, user *string, passwd *string, project *string, region *string ) ( o *Ostack ) {

	if host == nil || user == nil || passwd == nil {
		return
	}

	re  := regexp.MustCompile( "/[vV][1-9]+\\.{0,1}[0-9]*[/]{0,1}$"  )		// match version number, with or without .xxx, with or without trailing /, at end of string
	idx := re.FindStringIndex( *host )
	if idx != nil {
		i := idx[0]
		if i >= 0 {
			h := (*host)[0:i+1]
			host = &h
		}
	} else {
		if ! strings.HasSuffix( *host, "/" ) {
			h := *host + "/"
			host= &h
		}
	}

	o = &Ostack {
		passwd: passwd,
		user:	user,
		host:	host,
		project: project,
		aregion: region,
	}

	o.tok_isadmin = make( map[string]bool )

	return
}

/*
	Back compat constructor to default to nil region if it's not important to the user. 
*/
func Mk_ostack( host *string, user *string, passwd *string, project *string ) ( o *Ostack ) {
	return Mk_ostack_region( host, user, passwd, project, nil )
}

/*
	Duplicate the object adding the project name passed and then authorise to get a token
	and to pick up chost information for the project.
*/
func (o *Ostack) Dup(  project *string ) ( dup *Ostack, err error ) {

	dup = Mk_ostack_region( o.host, o.user, o.passwd, project, o.aregion )

	return
}

// -----------------------------------------------------------------------------------------


const (
	CVT_DASHES	bool = true			// convert dashes in json names to underbars
	NO_CVT		bool = false		// do not convert dashes in json names (data may be unusable)

	ANY			int = 0xff 			// host types for List_hosts() -- list all types
	COMPUTE		int = 0x01			// include compute hosts
	SCHEDULE	int = 0x02			// include list of scheduler hosts
	NETWORK		int = 0x04			// include list of network hosts
	CELLS		int	= 0x08			// include list of cells
	CONDUCTOR	int = 0x10			// include list of conductor hosts
	CERT		int = 0x20			// include list of certification hosts
	AUTH		int = 0x40			// include list of authorisation hosts
)

const (								// reset iota
	EP_COMPUTE	int = 0				// end point types for get_endpoint()
	EP_IDENTITY	 = iota
	EP_NETWORK	 = iota
)

// ---------- functions used by all other methods in this package -----------------------------------------------------

/*
	Sends a get request to openstack using the host in 'o' with the uri,  then extracts the resulting value if successful.
	The token, if not nil, is passed in the header. If the token appears to be one of the absurdly huge tokens (> 100 bytes)
	then we will use the md5 token that was computed during authorisation.  If openstack is returning short tokens, that 
	cannot be md5'd.
*/
func (o *Ostack) Send_req( method string, url *string, data *bytes.Buffer ) (jdata []byte, headers map[string][]string, err error) {
	var (
		req 	*http.Request;
		rsrc	*http.Client;		// request source	
	)
	
	jdata = nil;
	headers = nil;
	req, err = http.NewRequest( method, *url, data );
	if err != nil {
		fmt.Fprintf( os.Stderr, "error making request for %s to %s\n", method, *url );
		return
	}

	req.Header.Add( "Content-Type", "application/json" );
	if o.token != nil {											// authorisation won't have a token
		if len( *o.token ) > 100 {
			req.Header.Add( "X-Auth-Token", *o.small_tok );		// use compressed token
		} else {
			req.Header.Add( "X-Auth-Token", *o.token );
		}
	}

	rsrc = &http.Client{};
	resp, err := rsrc.Do( req )
	if err == nil {

		jdata, err = ioutil.ReadAll( resp.Body )
		resp.Body.Close( )

		headers = resp.Header;

		err = scanj4gook( jdata )				// quick scan to see if there are bad things in the json
	} else {
		fmt.Fprintf( os.Stderr, "ostack/Send_req: received err response %s\n", err );
	}

	return;
}

/*
	Returns true if this object matches the passed in ID string. 
*/
func (o *Ostack) Equals_id( id *string ) ( bool ) {
	if o == nil || o.project_id == nil {
		return false
	}

	return *o.project_id == *id
}

/*
	Returns true if this object matches the passed in name string. 
*/
func (o *Ostack) Equals_name( name *string ) ( bool ) {
	if o == nil || o.project == nil {
		return false
	}

	return *o.project == *name
}

/*
	Return the token that was generated for the object (testing)
*/
func (o *Ostack) Get_tok( ) ( string ) {
	if o == nil || o.token == nil {
		return ""
	}

	return *o.token
}

/*
	Test the expiration value in the set of credentials against the current
	time and return true if it is in the past.
*/
func (o *Ostack) Is_expired() ( bool ) {
	return o.expiry < time.Now().Unix()
}

/*
	Returns the user name associated with the credntial block.
*/
func (o *Ostack)  Get_user() ( *string ) {
	if o == nil || o.user == nil {
		return nil
	}

	return o.user
}

/* 
	Returns the project name and id
*/
func (o *Ostack) Get_project( ) ( name *string, id *string ) {
	if o == nil {
		return nil, nil
	}

	return o.project, o.project_id
}

/*
	Returns a string with some of the information that is being used to communicate with OpenStack.
*/
func (o *Ostack) To_str( ) ( s string ) {
	project := "none"

	if o == nil || o.host == nil || o.user == nil  {
		s = "invalid or missing openstack credentials"
	} else {
		if o.project != nil {
			project = *o.project
		}
		s = fmt.Sprintf( "ostack=<%s %s %s>", *o.host, *o.user, project );
	}
	return;
}

// ----- testing things mostly ----
func (o *Ostack) Get_token( ) ( *string ) {
	if o != nil {
		return o.token
	}
	return nil	
}
