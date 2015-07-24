
Gopkgs
======

Gopkgs is a collection of Go (golang) packages (a.k.a. libraries) which support other 
projects in this github organisation (e.g. Tegu), and that might be useful to 
other projects.


Currently, the pacages included:

###	arista		
Methods for interfacing with an Arista switch which is configured to use HTTP or HTTPs interface. 

###	bleater		
A level based logging package.

###	chkpt		
Provides an easy mechanism for creating dual-tumbler checkpoint files.

###	clike		
Some tools (atoi-ish) that behave in a Clib manner (not aborting if a
non-digit is encountered (ato* family) and add some extensions for
values with post-fixed units (e.g. 10GiB or 10G).

###	config		
A configuration file parser which provides for a section based file
and allows for inclusion of sub files. 

###	connman		
A TCP connection manager.

###	extcmd		
An external command interface which bundles the results (stdout/stderr)
into a managable structure for the caller.

###	ipc			
Interprocess communications support.  Provies a simple request/response
message block and some wrapper functions to easily send a message
on a channel.  Also provides a tickler function that can be started
and will send messages to a channel at prescribed times.

###	jsontools	
Tools which assist with the parsing and streasming json management.

###	ostack		
An interface to Openstack which provides authorisitaion, and general
queries making use of Openstack as a data source. 

###	security	
Support for generating self-signed certificates.

###	ssh_broker	
A broker which manages persistent SSH sessions with one or more 
hosts allowing for the remote execution of commands without
the session setup overhead needed if each call were executed via
a 'system()' like approach.

###	token		
Tokenising functions providing features like tokens with quotes
and embedded separators, unique token generation, token counting,
etc.


Go Package Doc
--------------
Running the Go package documentation tool on any of the packages in this source should 
generate the documentation needed to make use of these packages.  As an example

	`godoc gopkgs/token`

will generate the documentation on the token pacakge. 



