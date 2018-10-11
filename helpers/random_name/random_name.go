package random_name

import "github.com/cloudfoundry-incubator/cf-test-helpers/generator"

func BARARandomName(resource string) string {
	return generator.PrefixedRandomName("BARA", resource)
}
