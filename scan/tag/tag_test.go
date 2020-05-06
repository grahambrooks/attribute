package tag

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test(t *testing.T) {
	t.Run("No tags", func(t *testing.T) {
		tags, err := Parse([]string{})

		assert.NoError(t, err)
		assert.Equal(t, 0, len(tags.Contributor))
		assert.Equal(t, 0, len(tags.Repository))
	})

	t.Run("Tags assigned to the right key", func(t *testing.T) {
		tags, _ := Parse([]string{"Contributor:foo=bar", "Repository:key1=value10"})

		assert.Equal(t, 1, len(tags.Contributor))
		assert.Equal(t, 1, len(tags.Repository))
	})
	t.Run("Tag node names are case insensitive assigned to the right key", func(t *testing.T) {
		tags, _ := Parse([]string{"ConTriBuTor:foo=bar", "repository:key1=value10"})

		assert.Equal(t, 1, len(tags.Contributor))
		assert.Equal(t, 1, len(tags.Repository))
	})

	t.Run("Parsing a tag", func(t *testing.T) {
		tag, err := ParseTag("Contributor:foo=bar")

		assert.NoError(t, err)
		assert.Equal(t, "foo", tag.Key)
		assert.Equal(t, "bar", tag.Value)
		assert.Equal(t, "Contributor", tag.Node)
	})
}
