package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	log "github.com/sirupsen/logrus"
)

const (
	defaultOSFlavor = "m1.tiny"
	authurlVar      = "OS_AUTH_URL"
	usernameVar     = "OS_USERNAME"
	passwordVar     = "OS_PASSWORD"
	projectNameVar  = "OS_PROJECT_NAME"
	userDomainVar   = "OS_USER_DOMAIN_NAME"
)

func runOpenStack(args []string) {
	flags := flag.NewFlagSet("openstack", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run openstack [options] [name]\n\n", invoked)
		fmt.Printf("'name' is the name of an OpenStack image that has already been\n")
		fmt.Printf(" uploaded using 'linuxkit push'\n\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	authurlFlag := flags.String("authurl", "", "The URL of the OpenStack identity service, i.e https://keystone.example.com:5000/v3")
	flavorName := flags.String("flavor", defaultOSFlavor, "Instance size (flavor)")
	instanceName := flags.String("instancename", "", "Name of instance.  Defaults to the name of the image if not specified")
	networkID := flags.String("network", "", "The ID of the network to attach the instance to")
	keyName := flags.String("keyname", "", "The name of the SSH keypair to associate with the instance")
	passwordFlag := flags.String("password", "", "Password for the specified username")
	projectNameFlag := flags.String("project", "", "Name of the Project (aka Tenant) to be used")
	userDomainFlag := flags.String("domain", "Default", "Domain name")
	usernameFlag := flags.String("username", "", "Username with permissions to create an instance")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the name of the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	name := remArgs[0]

	if *instanceName == "" {
		*instanceName = name
	}

	authurl := getStringValue(authurlVar, *authurlFlag, "")
	password := getStringValue(passwordVar, *passwordFlag, "")
	projectName := getStringValue(projectNameVar, *projectNameFlag, "")
	userDomain := getStringValue(userDomainVar, *userDomainFlag, "")
	username := getStringValue(usernameVar, *usernameFlag, "")

	authOpts := gophercloud.AuthOptions{
		DomainName:       userDomain,
		IdentityEndpoint: authurl,
		Password:         password,
		TenantName:       projectName,
		Username:         username,
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		log.Fatalf("Failed to authenticate")
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		log.Fatalf("Unable to create Compute V2 client, %s", err)
	}

	network := servers.Network{
		UUID: *networkID,
	}

	var serverOpts servers.CreateOptsBuilder

	serverOpts = &servers.CreateOpts{
		FlavorName:    *flavorName,
		ImageName:     name,
		Name:          *instanceName,
		Networks:      []servers.Network{network},
		ServiceClient: client,
	}

	if *keyName != "" {
		serverOpts = &keypairs.CreateOptsExt{
			CreateOptsBuilder: serverOpts,
			KeyName:           *keyName,
		}
	}

	server, err := servers.Create(client, serverOpts).Extract()
	if err != nil {
		log.Fatalf("Unable to create server: %s", err)
	}

	servers.WaitForStatus(client, server.ID, "ACTIVE", 600)
	fmt.Printf("Server created, UUID is %s", server.ID)

}
