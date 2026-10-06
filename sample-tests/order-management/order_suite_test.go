package ordermanagement_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "github.com/reportportal/agent-go-ginkgo/pkg/rpagent"
)

func TestOrderManagement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Order Management Suite")
}
