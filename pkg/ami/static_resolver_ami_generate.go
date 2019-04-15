// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/weaveworks/eksctl/pkg/ami"
	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha4"

	. "github.com/dave/jennifer/jen"
)

func main() {
	fmt.Println("generating list of AMIs for the static resolvers")

	f := NewFile("ami")

	f.Comment("This file was generated by static_resolver_ami_generate.go; DO NOT EDIT.")
	f.Line()

	d := Dict{}

	client := newMultiRegionClient()

	for version := range ami.ImageSearchPatterns {
		versionImages := Dict{}
		for family := range ami.ImageSearchPatterns[version] {
			familyImages := Dict{}
			log.Printf("looking up %s/%s images", family, version)
			for class := range ami.ImageSearchPatterns[version][family] {
				classImages := Dict{}
				for _, region := range api.SupportedRegions() {
					p := ami.ImageSearchPatterns[version][family][class]
					log.Printf("looking up images matching %q in %q", p, region)
					image, err := ami.FindImage(client[region], p, family)
					if err != nil {
						log.Fatal(err)
					}
					classImages[Lit(region)] = Lit(image)
				}
				familyImages[Id(ami.ImageClasses[class])] = Values(classImages)
			}
			versionImages[Lit(family)] = Values(familyImages)
		}
		d[Lit(version)] = Values(versionImages)
	}

	f.Comment("StaticImages is a map that holds the list of AMIs to be used by for static resolution")

	f.Var().Id("StaticImages").Op("=").
		Map(String()).Map(String()).Map(Int()).Map(String()).String().Values(d)

	if err := f.Save("static_resolver_ami.go"); err != nil {
		log.Fatal(err.Error())
	}

}

func newSession(region string) *session.Session {
	config := aws.NewConfig()
	config = config.WithRegion(region)
	config = config.WithCredentialsChainVerboseErrors(true)

	// Create the options for the session
	opts := session.Options{
		Config:                  *config,
		SharedConfigState:       session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	return session.Must(session.NewSessionWithOptions(opts))
}

func newMultiRegionClient() map[string]*ec2.EC2 {
	clients := make(map[string]*ec2.EC2)
	for _, region := range api.SupportedRegions() {
		clients[region] = ec2.New(newSession(region))
	}
	return clients
}
