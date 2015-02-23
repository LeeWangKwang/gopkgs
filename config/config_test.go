// vi: sw=4 ts=4:

package config_test

import (
	"testing"
	"fmt"
	"os"

	"codecloud.web.att.com/gopkgs/config"
)

func TestConfig( t *testing.T ) {
	sects, err := config.Parse( nil, "test.cfg", false );

	if err != nil {
		fmt.Fprintf( os.Stderr, "parsing config file failed: %s\n", err );
		t.Fail();
		return;
	}

	for sname, smap := range sects {
		fmt.Fprintf( os.Stderr, "section: (%s) has %d items\n", sname, len( smap ) );
		for key, value := range smap {
			switch value.( type ) {
				case string:
					fmt.Fprintf( os.Stderr, "\t%s = %s\n", key, ( value.( string ) ) )

				case *string:
					fmt.Fprintf( os.Stderr, "\t%s = (%s)\n", key, *( value.( *string ) ) )

				case float64:
					fmt.Fprintf( os.Stderr, "\t%s = %v\n", key, value );

				default:
			}
		}
	} 

	smap := sects["default"]
	fmt.Fprintf( os.Stderr, "qfoo=== (%s)\n", *(smap["qfoo"].(*string)) ); 
	fmt.Fprintf( os.Stderr, "ffoo=== %8.2f\n", smap["ffoo"].(float64) ); 
	fmt.Fprintf( os.Stderr, "jfoo=== (%s)\n", *(smap["jfoo"].(*string)) ); 
}

/*
	test reading only as strings
*/
func TestStrings( t *testing.T ) {
	var (
		my_map map[string]map[string]*string;
		dup	string;
	)

	my_map = make( map[string]map[string]*string );
	my_map["default"] = make( map[string]*string );
	dup = "should be ovelaid by config file info";				// should be overridden
	my_map["default"]["ffoo"] = &dup;
	dup = "initial value, should exist after read";
	my_map["default"]["init-val"] = &dup;

fmt.Fprintf( os.Stderr, ">>>>> parsing\n" )
	sects, err := config.Parse2strs( my_map, "test.cfg" );

	if err != nil {
		fmt.Fprintf( os.Stderr, "parsing config file failed: %s\n", err );
		t.Fail();
		return;
	}

	for sname, smap := range sects {
		fmt.Fprintf( os.Stderr, "section: (%s) has %d items\n", sname, len( smap ) );
		for key, value := range smap {
			fmt.Fprintf( os.Stderr, "\t%s = (%s)\n", key, *value );
		}
	} 

	smap := sects["default"]									// can be referenced two different ways
	fmt.Fprintf( os.Stderr, "qfoo=== (%s)\n", *smap["qfoo"] ); 
	fmt.Fprintf( os.Stderr, "ffoo=== (%s)\n", *smap["ffoo"] ); 
	fmt.Fprintf( os.Stderr, "ffoo=== (%s)\n", *sects["default"]["ffoo"] ); 
}

