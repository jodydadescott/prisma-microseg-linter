package example

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jodydadescott/prisma-microseg-linter/prisma"
)

func databaseConfig1() *prisma.Config {

	// This is a database. Allow it access to other databases inside the same namespace on the same cluster
	// and on the same namespace in different clusters

	// Source
	sub := prisma.NewSubjectObjectBuilder().New()
	sub.Add("@org:tenant=trash")
	sub.Add("@org:cloudaccount=trash")
	sub.Add("@org:group=trash")
	sub.Add("@org:group=trash")
	sub.Add("app=database")

	out := prisma.NewSubjectObjectBuilder()

	// Destination
	local := out.New()
	local.Add("@org:tenant=841735782980352000")
	local.Add("@org:cloudaccount=cloud1")
	local.Add("@org:group=cluster1")
	local.Add("@org:kubernetes=database")
	local.Add("app=database")

	remote1 := out.New()
	remote1.Add("@org:tenant=841735782980352000")
	remote1.Add("@org:cloudaccount=cloud1")
	remote1.Add("@org:group=cluster2")
	remote1.Add("@org:kubernetes=database")
	remote1.Add("app=database")

	remote2 := out.New()
	remote2.Add("@org:tenant=841735782980352000")
	remote2.Add("@org:cloudaccount=cloud2")
	remote2.Add("@org:group=cluster1")
	remote2.Add("@org:kubernetes=database")
	remote2.Add("app=database")

	object := out.Build()

	outgoingRule := prisma.NewRule().SetAction(prisma.TrafficActionAllow).SetObject(object)
	incomingRule := prisma.NewRule().SetAction(prisma.TrafficActionAllow).SetObject(object)

	policy := prisma.NewNetworkrulesetpolicy("ruleset1").AddOutgoingRule(outgoingRule).AddIncomingRule(incomingRule)
	policy.SetSubject(sub.Build())

	return prisma.NewConfig("config1").AddNetworkrulesetpolicy(policy)
}

func appServerConfig1() *prisma.Config {

	// This is an app. Its allowed to talk to the database and other apps

	// Source
	sub := prisma.NewSubjectObjectBuilder().New()
	sub.Add("@org:tenant=trash")
	sub.Add("@org:cloudaccount=trash")
	sub.Add("@org:group=trash")
	sub.Add("@org:group=trash")
	sub.Add("app=app")

	out := prisma.NewSubjectObjectBuilder()

	// Destination
	localOut := out.New()
	localOut.Add("@org:tenant=841735782980352000")
	localOut.Add("@org:cloudaccount=cloud1")
	localOut.Add("@org:group=cluster1")
	localOut.Add("@org:kubernetes=app")
	localOut.Add("app=app")

	remoteOut := out.New()
	remoteOut.Add("@org:tenant=841735782980352000")
	remoteOut.Add("@org:cloudaccount=cloud1")
	remoteOut.Add("@org:group=cluster2")
	remoteOut.Add("@org:kubernetes=app")
	remoteOut.Add("app=app")

	database1Out := out.New()
	database1Out.Add("@org:tenant=841735782980352000")
	database1Out.Add("@org:cloudaccount=cloud1")
	database1Out.Add("@org:group=cluster1")
	database1Out.Add("@org:kubernetes=database")
	database1Out.Add("app=database")

	database2Out := out.New()
	database2Out.Add("@org:tenant=841735782980352000")
	database2Out.Add("@org:cloudaccount=cloud1")
	database2Out.Add("@org:group=cluster2")
	database2Out.Add("@org:kubernetes=database")
	database2Out.Add("app=database")

	in := prisma.NewSubjectObjectBuilder()

	// Destination
	localIn := in.New()
	localIn.Add("@org:tenant=841735782980352000")
	localIn.Add("@org:cloudaccount=cloud1")
	localIn.Add("@org:group=cluster1")
	localIn.Add("@org:kubernetes=app")
	localIn.Add("app=app")

	remoteIn1 := out.New()
	remoteIn1.Add("@org:tenant=841735782980352000")
	remoteIn1.Add("@org:cloudaccount=cloud1")
	remoteIn1.Add("@org:group=cluster2")
	remoteIn1.Add("@org:kubernetes=app")
	remoteIn1.Add("app=app")

	remoteIn2 := out.New()
	remoteIn2.Add("@org:tenant=841735782980352000")
	remoteIn2.Add("@org:cloudaccount=cloud2")
	remoteIn2.Add("@org:group=cluster2")
	remoteIn2.Add("@org:kubernetes=app")
	remoteIn2.Add("app=app")

	outgoingRule := prisma.NewRule().SetAction(prisma.TrafficActionAllow).SetObject(out.Build())
	incomingRule := prisma.NewRule().SetAction(prisma.TrafficActionAllow).SetObject(in.Build())

	policy := prisma.NewNetworkrulesetpolicy("ruleset1").AddOutgoingRule(outgoingRule).AddIncomingRule(incomingRule)
	policy.SetSubject(sub.Build())

	return prisma.NewConfig("config2").AddNetworkrulesetpolicy(policy)
}

// func orpahConfig1() *prisma.Config {

// 	// This refers to something that does not exist

// 	// Source
// 	sub := prisma.NewSubjectObjectBuilder().New()
// 	sub.Add("@org:tenant=trash")
// 	sub.Add("@org:cloudaccount=trash")
// 	sub.Add("@org:group=trash")
// 	sub.Add("@org:group=trash")
// 	sub.Add("app=app")

// 	out := prisma.NewSubjectObjectBuilder()

// 	// Destination
// 	localOut := out.New()
// 	localOut.Add("@org:tenant=841735782980352000")
// 	localOut.Add("@org:cloudaccount=fake_cloud")

// 	remoteOut := out.New()
// 	remoteOut.Add("@org:tenant=841735782980352000")
// 	remoteOut.Add("@org:cloudaccount=cloud1")
// 	remoteOut.Add("@org:group=cluster2")
// 	remoteOut.Add("@org:kubernetes=app")
// 	remoteOut.Add("app=app")

// 	database1Out := out.New()
// 	database1Out.Add("@org:tenant=841735782980352000")
// 	database1Out.Add("@org:cloudaccount=cloud1")
// 	database1Out.Add("@org:group=cluster1")
// 	database1Out.Add("@org:kubernetes=database")
// 	database1Out.Add("app=database")

// 	database2Out := out.New()
// 	database2Out.Add("@org:tenant=841735782980352000")
// 	database2Out.Add("@org:cloudaccount=cloud1")
// 	database2Out.Add("@org:group=cluster2")
// 	database2Out.Add("@org:kubernetes=database")
// 	database2Out.Add("app=database")

// 	in := prisma.NewSubjectObjectBuilder()

// 	// Destination
// 	localIn := in.New()
// 	localIn.Add("@org:tenant=841735782980352000")
// 	localIn.Add("@org:cloudaccount=cloud1")
// 	localIn.Add("@org:group=cluster1")
// 	localIn.Add("@org:kubernetes=app")
// 	localIn.Add("app=app")

// 	remoteIn1 := out.New()
// 	remoteIn1.Add("@org:tenant=841735782980352000")
// 	remoteIn1.Add("@org:cloudaccount=cloud1")
// 	remoteIn1.Add("@org:group=cluster2")
// 	remoteIn1.Add("@org:kubernetes=app")
// 	remoteIn1.Add("app=app")

// 	remoteIn2 := out.New()
// 	remoteIn2.Add("@org:tenant=841735782980352000")
// 	remoteIn2.Add("@org:cloudaccount=cloud2")
// 	remoteIn2.Add("@org:group=cluster2")
// 	remoteIn2.Add("@org:kubernetes=app")
// 	remoteIn2.Add("app=app")

// 	outgoingRule := prisma.NewRule().SetAction(prisma.TrafficActionAllow).SetObject(out.Build())
// 	incomingRule := prisma.NewRule().SetAction(prisma.TrafficActionAllow).SetObject(in.Build())

// 	policy := prisma.NewNetworkrulesetpolicy("ruleset1").AddOutgoingRule(outgoingRule).AddIncomingRule(incomingRule)
// 	policy.SetSubject(sub.Build())

// 	return prisma.NewConfig("config2").AddNetworkrulesetpolicy(policy)
// }

func writePrismaConfig(dirname, filename string, prismaConfig *prisma.Config) error {

	err := os.MkdirAll(dirname, os.ModePerm)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(&prismaConfig)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(dirname+"/"+filename, data, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func writeNotes(dirname, data string) error {

	data = strings.TrimSpace(data)

	err := os.MkdirAll(dirname, os.ModePerm)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(dirname+"/notes.txt", []byte(data), os.ModePerm); err != nil {
		return err
	}

	return nil
}

func appA(dirname string) error {

	notes := `
This App will fail validation because the tags are incorrect. Running sanatize will
correct these tags. 

Note: When calling sanatize and validate, sanatize will run first and will fix issues
such as labels. Hence if you wish to see these errors you will need to run validate
without the sanatize option. If you have already ran sanatize run the restore.sh script
and then run with the validate option only.
`

	dirname = dirname + "/app_a"

	databaseConfig1 := databaseConfig1()

	appServerConfig1 := appServerConfig1()

	if err := writeNotes(dirname, notes); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster1/database", "policy1.yaml", databaseConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster2/database", "policy1.yaml", databaseConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster1/app", "policy1.yaml", appServerConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster2/app", "policy2.yaml", appServerConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud2/cluster1/database", "policy1.yaml", databaseConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud2/cluster2/database", "policy1.yaml", databaseConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud2/cluster1/app", "policy1.yaml", appServerConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud2/cluster2/app", "policy2.yaml", appServerConfig1); err != nil {
		return err
	}

	return nil
}

func appB(dirname string) error {

	notes := `
This App will fail validation because the tags are incorrect and because there are
references to a non-existent cloud. Running sanatize will correct the labels but it
will not be able to correct the non-existent cloud issue.

Note: When calling sanatize and validate, sanatize will run first and will fix issues
such as labels. Hence if you wish to see these errors you will need to run validate
without the sanatize option. If you have already ran sanatize run the restore.sh script
and then run with the validate option only.
`

	dirname = dirname + "/app_b"

	databaseConfig1 := databaseConfig1()

	appServerConfig1 := appServerConfig1()

	if err := writeNotes(dirname, notes); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster1/database", "policy1.yaml", databaseConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster2/database", "policy1.yaml", databaseConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster1/app", "policy1.yaml", appServerConfig1); err != nil {
		return err
	}

	if err := writePrismaConfig(dirname+"/841735782980352000/cloud1/cluster2/app", "policy2.yaml", appServerConfig1); err != nil {
		return err
	}

	return nil
}

// Write creates a directory structure of child directories and files.
func Write(dirname string) error {

	if _, err := os.Stat(dirname); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exist; aborting", dirname)
	}

	if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
		return err
	}

	if err := appA(dirname); err != nil {
		return err
	}

	if err := appB(dirname); err != nil {
		return err
	}

	return nil
}
