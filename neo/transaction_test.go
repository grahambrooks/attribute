package neo

import (
	"github.com/grahambrooks/attribute/scan/tag"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQueryConstruction(t *testing.T) {
	t.Run("Tag parameter construction", func(t *testing.T) {
		tags := make([]tag.Tag, 0)

		p := makeTagParams(tags)
		assert.Equal(t, "", p)
	})

	t.Run("Single tag parameter", func(t *testing.T) {
		tags := []tag.Tag{{
			Node:  "",
			Key:   "foo",
			Value: "bar",
		}}

		p := makeTagParams(tags)
		assert.Equal(t, ", foo: $foo", p)
	})

	t.Run("Multiple tag parameter", func(t *testing.T) {
		tags := []tag.Tag{{
			Node:  "",
			Key:   "one",
			Value: "bar",
		},
			{
				Node:  "",
				Key:   "two",
				Value: "bar",
			}}

		p := makeTagParams(tags)
		assert.Equal(t, ", one: $one, two: $two", p)
	})
}

func TestMakeStatement(t *testing.T) {
	t.Run("No Tags", func(t *testing.T) {
		statement := makeStatement(`MERGE (n:Repository { name: $name, origin: $origin%s }) RETURN n`, []tag.Tag{})

		assert.Equal(t, statement, `MERGE (n:Repository { name: $name, origin: $origin }) RETURN n`)
	})
	t.Run("With tags", func(t *testing.T) {
		tags := []tag.Tag{{
			Node:  "",
			Key:   "one",
			Value: "bar",
		},
			{
				Node:  "",
				Key:   "two",
				Value: "bar",
			}}

		statement := makeStatement(`MERGE (n:Repository { name: $name, origin: $origin%s }) RETURN n`, tags)

		assert.Equal(t, statement, `MERGE (n:Repository { name: $name, origin: $origin, one: $one, two: $two }) RETURN n`)
	})
}
