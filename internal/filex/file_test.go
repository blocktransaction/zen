package filex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile(t *testing.T) {
	assert := assert.New(t)

	files, _ := ListFiles("../../", "*.json", true, true)
	assert.Empty(files, "file found")

}
