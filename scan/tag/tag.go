package tag

import (
	"fmt"
	"regexp"
	"strings"
)

type Tag struct {
	Node  string
	Key   string
	Value string
}

type Tags struct {
	Contributor []Tag
	Repository  []Tag
}

func ParseTag(spec string) (*Tag, error) {
	r := regexp.MustCompile(`([a-zA-Z]+):([a-zA-Z-_0-9]+)=([a-zA-Z-_0-9]+)`)

	matches := r.FindStringSubmatch(spec)

	fmt.Printf("matches %v", matches)

	if len(matches) == 4 {
		return &Tag{
			Node:  matches[1],
			Key:   matches[2],
			Value: matches[3],
		}, nil
	} else {
		return nil, fmt.Errorf("no match: unable to parse tag %s", spec)
	}
}

func Parse(tagArgs []string) (*Tags, error) {
	tags := Tags{}
	for _, tag := range tagArgs {
		t, err := ParseTag(tag)
		if err == nil {
			switch {
			case strings.EqualFold("contributor", t.Node):
				tags.Contributor = append(tags.Contributor, *t)
			case strings.EqualFold("repository", t.Node):
				tags.Repository = append(tags.Repository, *t)
			default:
				return nil, fmt.Errorf("Node name not recognized %s", t.Node)
			}
		} else {
			return nil, err
		}
	}
	return &tags, nil
}
