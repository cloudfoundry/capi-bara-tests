package random_name

import "github.com/cloudfoundry/cf-test-helpers/v2/generator"

func BARARandomName(resource string) string {
	return generator.PrefixedRandomName("BARA", resource)
}
