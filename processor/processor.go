package processor

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v3"

	"github.com/jodydadescott/prisma-microseg-linter/prisma"
)

// Namespace : Prisma supports the concept of namespaces. Namespaces are hierarchical and recursive
// so that each namespace may contain zero or more configurations and child namespaces.
// This is very much analogous to a computer file system. With the notable exception of
// the root namespace each namespace has one and only one parent.
//
// Prisma is opinionated in regards to the layout of the namespace structure. The strucutre is a follows;
// Root: This namespace is not customer owned
// Tenant: This is the highest level customer namespace
// Cloud (or Cloud Account): This represents a cloud provider
// Group: This represents either a Kubernetes cluster or a logical grouping of VMs.
// Kubernetes: This only applies if the Group is of type Kubernetes and corresponds to the Kubernetes namespace
//
// This utility expects to injest a directory strucutre that mimics the namespace strucutre defined above with the
// following caveats:
// 1) With the exception of the root directory, the name of each directory should correspond to the name in Prisma
// with no deviation. For example the root directory should contain one or more directories named the same as the
// Prisma Tenant ID.
// 2) The root directory should only contain one or more tenant directores and it should NOT contain any configuration
// files. The rationalization for this is that the customer does not own the directory above tenant and may not
// modify it.
type Namespace struct {
	rootNamespace                    *Namespace
	childMap                         map[string]*Namespace
	level                            metaLevel
	tenant, cloud, group, kubernetes string
	fileMap                          map[string]*file
	path, rpath, label               string
	restore                          string
	verbose                          bool
}

// file represents a YAML config file. We are not exposing it outside of this package
type file struct {
	parent             *Namespace // a file needs access to its parent namespace (directory)
	filename           string
	path, rpath, label string
	prismaConfig       *prisma.Config
}

// NewNamespace returns a new Namespace at the Root level. This Namespace base directory can be configured
// as an option or the current working directory will be used. If there are errors a collection of wrapped
// errors will be returned. If there are no errors nil will be returned.
func NewNamespace(path string, verbose bool) (*Namespace, error) {

	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	t := &Namespace{
		path:     path,
		childMap: make(map[string]*Namespace),
		level:    metaLevelRoot,
		verbose:  verbose,
	}

	t.rootNamespace = t

	// We are calling read() which will return the error (or nil). Note that read is recursive
	// so while the first call will read the root directory as specified here any directory that
	// exist below will result in another call to read (hence recursive)
	return t, t.read()
}

// This call is made for every directory that exist below root
func (t *Namespace) newNamespace(name string) *Namespace {

	label := ""
	if t.label == "" {
		label = name
	} else {
		label = t.label + ":" + name
	}

	rpath := ""
	if t.rpath == "" {
		rpath = name
	} else {
		rpath = t.rpath + "/" + name
	}

	meta := &Namespace{
		verbose:       t.verbose,
		tenant:        t.tenant,
		cloud:         t.cloud,
		group:         t.group,
		kubernetes:    t.kubernetes,
		path:          t.path + "/" + name,
		rpath:         rpath,
		label:         label,
		childMap:      make(map[string]*Namespace),
		fileMap:       make(map[string]*file),
		rootNamespace: t.rootNamespace,
	}

	switch t.level {

	case metaLevelRoot:
		meta.level = metaLevelTenant
		meta.tenant = name
		return meta

	case metaLevelTenant:
		meta.level = metaLevelCloud
		meta.cloud = name
		return meta

	case metaLevelCloud:
		meta.level = metaLevelGroup
		meta.group = name
		return meta

	case metaLevelGroup:
		meta.level = metaLevelKubernetes
		meta.kubernetes = name
		return meta
	}

	panic("depth recursion limit exceeded")
}

// Sanatize iterates all of the configurations and configures the subject based on
// the location within the directory strucutre and writes the file back to disk in
// the same location as the original. Note that this function is recursive.
func (t *Namespace) Sanatize() error {

	var errors *multierror.Error

	for _, x := range t.childMap {
		if err := x.Sanatize(); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	// Root does not have any files!
	if t.level != metaLevelRoot {
		for _, x := range t.fileMap {
			if err := x.sanatize(); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}

	if t.restore != "" {
		restoreFile := t.path + "/restore.sh"

		if t.verbose {
			log.Printf("writing restore file \"%s\"", restoreFile)
		}

		if err := ioutil.WriteFile(restoreFile, []byte(restoreScript+t.restore), os.ModePerm); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}

// Validate verifies that the policy objects reference valid subjects. For example
// if a policy subject is group=a, cloud=b, tenant=c and no subject with those tags
// exist then an error will we returned. Errors are collated. Note that this function is
// recursive.
func (t *Namespace) Validate() error {

	var errors *multierror.Error

	for _, meta := range t.childMap {
		if err := meta.Validate(); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	for _, meta := range t.fileMap {
		if err := meta.validate(); err != nil {
			errors = multierror.Append(errors, err)
		}
	}

	return errors.ErrorOrNil()
}

// This call is made for each matching configuration file found in any
// directory with the notable exception of root.
func (t *Namespace) newFile(filename string) *file {

	return &file{
		parent:   t,
		filename: filename,
		path:     t.path + "/" + filename,
		rpath:    t.rpath + "/" + filename,
		label:    t.label + ":" + filename,
	}
}

// This function iterates the directories and files inside the current namespace. For each directory a new namespace
// is created which will also result in another call to read() hence it is recursive. For each matching file a new
// "file" will be created and stored in its parent namespace. Note that if ANY matching files are located in the root
// will cause an error to be returned. This is because the root level namespace is not permitted to have configuration.
func (t *Namespace) read() error {

	// This checks to see if a given filename has a desired extension such as .yaml
	extensionCheck := func(filename string) bool {
		for _, match := range configFileExtensions {
			if strings.HasSuffix(filename, match) {
				return true
			}
		}
		return false
	}

	var errors *multierror.Error

	files, err := ioutil.ReadDir(t.path)
	if err != nil {
		return err
	}

	for _, file := range files {

		fileName := file.Name()

		if file.IsDir() {

			if fileName == ".original" {
				if t.verbose {
					log.Printf("ignoring directory \"%s\"", t.path+"/"+fileName)
				}
				continue
			}

			log.Printf("loading directory \"%s\"", t.path+"/"+fileName)

			if t.level == metaLevelKubernetes {
				errors = multierror.Append(errors, fmt.Errorf("Directory \"%s\" has child directories; this is not valid", t.path+"/"+fileName))
				continue
			}

			newMeta := t.newNamespace(fileName)
			t.childMap[fileName] = newMeta

			if err := newMeta.read(); err != nil {
				errors = multierror.Append(errors, err)
			}

		} else {

			if extensionCheck(fileName) {

				if t.level == metaLevelRoot {
					log.Printf("file \"%s\" should NOT exist here", fileName)
					errors = multierror.Append(errors, fmt.Errorf("File \"%s\" exist in root directory; this is not valid", t.path+"/"+fileName))

				} else {

					log.Printf("injesting file \"%s\"", fileName)

					metaFile := t.newFile(fileName)
					t.fileMap[fileName] = metaFile

					if err := metaFile.load(); err != nil {
						errors = multierror.Append(errors, err)
					}

				}
			} else {

				if t.verbose {
					log.Printf("ignoring file \"%s\"", fileName)
				}

			}
		}

	}

	return errors.ErrorOrNil()
}

func (t *Namespace) addRestore(original, new string) {
	t.restore = t.restore + fmt.Sprintf("cp %s %s\n", original, new)
}

// This takes in a string and returns two strings using the '=' character as the split
// If there is no '=' char then an empty value will be returned
func keyValueSplit(keyValuePair string) (string, string) {
	split := strings.Split(keyValuePair, "=")
	key := split[0]
	value := ""
	if len(split) > 1 {
		value = split[1]
	}

	return key, value
}

// this verifies that the policy objects reference valid subjects. For example
// if a policy subject is group=a, cloud=b, tenant=c and no subject with those tags
// exist then an error will we returned. Errors are collated. This is called by the
// Namespace when the Validate() function is called.
func (t *file) validate() error {

	if t.prismaConfig == nil {
		return fmt.Errorf("missing prismaConfig in file %s", t.path)
	}

	if t.prismaConfig.Data == nil {
		return fmt.Errorf("missing prismaConfig.Data in file %s", t.path)
	}

	var errors *multierror.Error

	top := t.parent.rootNamespace.path

	tenantCheck := func(rule, tenant string) *Namespace {

		if tenant == "" {
			errors = multierror.Append(errors, fmt.Errorf("subject %s does not have a tenant tag in file \"%s\"", rule, t.path))
			return nil
		}

		ns := t.parent.rootNamespace.childMap[tenant]
		if ns == nil {
			errors = multierror.Append(errors, fmt.Errorf("subject %s in file \"%s\" references non existent tenant namespace %s", rule, t.path, top+"/"+tenant))
		}
		return ns
	}

	cloudCheck := func(rule, tenant, cloud string) *Namespace {

		parentNS := tenantCheck(rule, tenant)
		if parentNS == nil {
			return nil
		}

		if cloud == "" {
			errors = multierror.Append(errors, fmt.Errorf("subject %s does not have a cloud tag in file \"%s\"", rule, t.path))
			return nil
		}

		ns := parentNS.childMap[cloud]
		if ns == nil {
			errors = multierror.Append(errors, fmt.Errorf("subject %s in file \"%s\" references non existent cloud namespace %s/%s", rule, t.path, top+"/"+tenant, cloud))
		}
		return ns
	}

	groupCheck := func(rule, tenant, cloud, group string) *Namespace {

		parentNS := cloudCheck(rule, tenant, cloud)
		if parentNS == nil {
			return nil
		}

		if group == "" {
			errors = multierror.Append(errors, fmt.Errorf("subject %s does not have a group tag in file \"%s\"", rule, t.path))
			return nil
		}

		ns := parentNS.childMap[group]
		if ns == nil {
			errors = multierror.Append(errors, fmt.Errorf("subject %s in file \"%s\" references non existent group namespace %s/%s/%s", rule, t.path, top+"/"+tenant, cloud, group))
		}
		return ns
	}

	kubernetesCheck := func(rule, tenant, cloud, group, kubernetes string) {

		parentNS := groupCheck(rule, tenant, cloud, group)
		if parentNS == nil {
			return
		}

		if kubernetes == "" {
			errors = multierror.Append(errors, fmt.Errorf("subject %s does not have a kubernetes tag in file \"%s\"", rule, t.path))
			return
		}

		ns := parentNS.childMap[kubernetes]
		if ns == nil {
			errors = multierror.Append(errors, fmt.Errorf("subject %s in file \"%s\" references non existent kubernetes namespace %s/%s/%s/%s", rule, t.path, top+"/"+tenant, cloud, group, kubernetes))
		}
	}

	nsCheck := func(rule, tenant, cloud, group, kubernetes string) {

		if kubernetes != "" {
			kubernetesCheck(rule, tenant, cloud, group, kubernetes)
			return
		}

		if group != "" {
			groupCheck(rule, tenant, cloud, group)
			return
		}

		if cloud != "" {
			cloudCheck(rule, tenant, cloud)
			return
		}

		tenantCheck(rule, tenant)
	}

	// This function verifies that every object references valid subjects.
	objectCheck := func(rule string, object []string) {

		tenant := ""
		cloud := ""
		group := ""
		kubernetes := ""

		// Note the hiearchy is tenant->cloud->group->kubernetes
		// The tags may be in any order so we iterate them and pick out the ones we care about
		// (those just listed). The rules are as follows:
		// 1) The tag with the key 'tenant' MUST always be present
		// 2) If any other tag is present its hiearchial parent MUST also be present. For example if
		// 'group' is set then 'cloud' must also be set.

		for _, keyValuePair := range object {

			key, value := keyValueSplit(keyValuePair)

			switch key {

			case "@org:tenant":
				tenant = value

			case "@org:cloudaccount":
				cloud = value

			case "@org:group":
				group = value

			case "@org:kubernetes":
				kubernetes = value

			}

		}

		nsCheck(rule, tenant, cloud, group, kubernetes)
	}

	// This function verifies that each rules object references valid subjects.
	ruleCheck := func(rule string, rules []*prisma.Rule) {

		for _, rules := range rules {
			if rules.Object == nil || len(rules.Object) <= 0 {
				log.Printf("there are no OutgoingRules in file \"%s\"", t.path)
			} else {
				for _, object := range rules.Object {
					objectCheck(rule, object)
				}
			}
		}

	}

	if t.prismaConfig.Data.Networkrulesetpolicies == nil || len(t.prismaConfig.Data.Networkrulesetpolicies) <= 0 {
		log.Printf("there are no Networkrulesetpolicies in file \"%s\"", t.path)
	} else {

		log.Printf("processing Networkrulesetpolicies in file \"%s\"", t.path)

		for _, ruleset := range t.prismaConfig.Data.Networkrulesetpolicies {

			if ruleset == nil {
				log.Printf("there are no rules in file \"%s\"", t.path)
			} else {
				ruleCheck("OutgoingRules", ruleset.OutgoingRules)
				ruleCheck("IncomingRules", ruleset.IncomingRules)
			}

		}

	}

	if t.prismaConfig.Label != t.label {
		errors = multierror.Append(errors, fmt.Errorf("label \"%s\" should be \"%s\" in file \"%s\"", t.prismaConfig.Label, t.label, t.path))
	}

	return errors.ErrorOrNil()
}

// This iterates the prisma config and configures the subject based on
// the location within the directory strucutre and writes the file back to disk in
// the same location as the original. This function is used by the Namespace func Sanatize
func (t *file) sanatize() error {

	// Subject
	// 1) Flatten the subject from [][]string to []string
	// 2) Remove any tags with the the key prefix of '@org'
	// 3) Add in the '@org' entries based on the current level

	// - - "@org:cloudaccount="
	//   - "@org:group={{.Values.org.group}}"
	//   - "@org:tenant={{.Values.org.tenant}}"
	//   - "@org:kubernetes=knoxville"
	//   - app=backend

	if t.prismaConfig == nil {
		return nil
	}

	ruleSetHash := func(policies []*prisma.Networkrulesetpolicy) string {

		result := ""
		for _, ruleset := range policies {
			result = result + ruleset.String()
		}

		return result
	}

	preRuleSetPolicyString := ruleSetHash(t.prismaConfig.Data.Networkrulesetpolicies)

	for _, ruleset := range t.prismaConfig.Data.Networkrulesetpolicies {

		var newSubs []string

		addTenant := func() {
			newSubs = append(newSubs, "@org:tenant="+t.parent.tenant)
		}

		addCloud := func() {
			addTenant()
			newSubs = append(newSubs, "@org:cloudaccount="+t.parent.cloud)
		}

		addGroup := func() {
			addCloud()
			newSubs = append(newSubs, "@org:group="+t.parent.group)
		}

		addKubernetes := func() {
			addGroup()
			newSubs = append(newSubs, "@org:kubernetes="+t.parent.kubernetes)
		}

		for _, subject := range ruleset.Subject {
			for _, sub := range subject {
				if !strings.HasPrefix(sub, "@org:") {
					newSubs = append(newSubs, sub)
				}
			}
		}

		switch t.parent.level {

		case metaLevelTenant:
			addTenant()

		case metaLevelCloud:
			addCloud()

		case metaLevelGroup:
			addGroup()

		case metaLevelKubernetes:
			addKubernetes()

		}

		var newSubSubs [][]string
		newSubSubs = append(newSubSubs, newSubs)

		ruleset.Subject = newSubSubs
	}

	dirty := false

	postRuleSetPolicyString := ruleSetHash(t.prismaConfig.Data.Networkrulesetpolicies)

	if preRuleSetPolicyString != postRuleSetPolicyString {
		log.Printf("tags updated in file \"%s\"", t.path)
		dirty = true
	} else {
		if t.parent.verbose {
			log.Printf("no changes to tags updated file \"%s\"", t.path)
		}
	}

	oldLabel := t.prismaConfig.Label

	if oldLabel != t.label {
		t.prismaConfig.Label = t.label
		log.Printf("label changed from \"%s\" to \"%s\" in file \"%s\"", oldLabel, t.label, t.path)
		dirty = true
	} else {
		if t.parent.verbose {
			log.Printf("no change to label in file \"%s\"", t.path)
		}
	}

	//	filenameWithoutExt := strings.TrimRight(t.path, filepath.Ext(t.path))

	if dirty {

		originalDirPath := t.parent.path + "/.original"
		originalDirRPath := t.parent.rpath + "/.original"

		if err := os.MkdirAll(originalDirPath, os.ModePerm); err != nil {
			return err
		}

		originalFilePath := originalDirPath + "/" + t.filename
		originalFileRPath := originalDirRPath + "/" + t.filename

		if err := os.Rename(t.path, originalFilePath); err != nil {
			return fmt.Errorf("failed to rename file %s to %s: %w", t.path, originalFilePath, err)
		}

		t.parent.rootNamespace.addRestore(originalFileRPath, t.rpath)

		// if err := os.Rename(t.path, filenameWithoutExt+".backup"); err != nil {
		// 	return fmt.Errorf("failed to rename file %s : %w", t.path, err)
		// }

		data, err := yaml.Marshal(&t.prismaConfig)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(t.path, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s : %w", t.path, err)
		}
		log.Printf("file \"%s\" updated", t.path)
	} else {
		if t.parent.verbose {
			log.Printf("no change to file \"%s\"", t.path)
		}
	}

	return nil
}

// this loads a prisma config file into the file
func (t *file) load() error {

	b, err := ioutil.ReadFile(t.path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(b, &t.prismaConfig); err != nil {
		return err
	}

	return nil
}
