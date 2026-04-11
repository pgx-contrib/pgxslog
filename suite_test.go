package pgxslog_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPgxslog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pgxslog Suite")
}
