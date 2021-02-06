package tests

import (
	"github.com/owenrumney/squealer/internal/app/squealer/config"
	"github.com/owenrumney/squealer/internal/app/squealer/scan"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRepoEndToEnd(t *testing.T) {

	scanner, err := scan.NewGitScanner(scan.NewScannerConfig(testPath, true, config.DefaultConfig()))
	assert.NoError(t, err)

	err = scanner.Scan()
	assert.NoError(t, err)

	metrics := scanner.GetMetrics()

	assert.Equal(t, int32(3), metrics.CommitsProcessed)
	assert.Equal(t, int32(8), metrics.TransgressionsFound)
	assert.Equal(t, int32(0), metrics.TransgressionsIgnored)
	assert.Equal(t, int32(4), metrics.TransgressionsReported)
}
