package fasthttpprometheus

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
)

type trieSuite struct {
	suite.Suite

	paths [5]map[string]string
	node  *node
	trie  *node
}

func TestTrieSuite(t *testing.T) {
	suite.Run(t, &trieSuite{})
}

func (s *trieSuite) SetupTest() {
	s.paths = [5]map[string]string{
		{
			"metric_name": "user",
			"path":        "/user/:id",
		},
		{
			"metric_name": "user_some_method_one",
			"path":        "/user/:id/some-method-one",
		},
		{
			"metric_name": "user_some_method_two",
			"path":        "/user/:id/some-method-two",
		},
		{
			"metric_name": "ping",
			"path":        "/ping",
		},
		{
			"metric_name": "article_some_action",
			"path":        "/article/some-action/:id",
		},
	}

	s.node = new(node)
	s.trie = &node{
		children: []*node{
			{
				path: "user",
				children: []*node{
					{
						path: ":id",
						children: []*node{
							{
								path: "some-method-one",
							},
							{
								path: "some-method-two",
							},
						},
					},
				},
			},
			{
				path: "ping",
			},
			{
				path: "article",
				children: []*node{
					{
						path: "some-action",
						children: []*node{
							{
								path: ":id",
							},
						},
					},
				},
			},
		},
	}
}

func (s *trieSuite) TestAppendChild() {
	var metricName string
	s.node.appendChild(8, "article", s.paths[4]["path"], &metricName)

	s.Equal(&node{
		children: []*node{
			{
				path: "article",
				children: []*node{
					{
						path: "some-action",
						children: []*node{
							{
								path: ":id",
							},
						},
					},
				},
			},
		},
	}, s.node)
}

func (s *trieSuite) TestAddPath() {
	for _, value := range s.paths {
		var metricName string
		leaf := s.node.addPath(value["path"], &metricName)

		ss := strings.Split(value["path"], "/")
		s.Equal(&node{path: ss[len(ss)-1]}, leaf)
		s.Equal(value["metric_name"], metricName)
	}

	s.Equal(s.trie, s.node)
}

func (s *trieSuite) TestLoopChildren() {
	metrics := map[string]prometheus.Counter{
		metricTypeTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "test",
			Name:      "test_total",
		}),
		metricTypeFailure: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "test",
			Name:      "test_failure",
		}),
	}

	trie := &node{
		path: "some-action",
		children: []*node{
			{
				path:    ":id",
				metrics: metrics,
			},
		},
	}

	node := trie.loopChildren(
		"/some-action/:id",
		12,
		func(child *node) *node {
			return child
		},
	)

	s.Equal(metrics, node.metrics)
}

func (s *trieSuite) TestGetLeaf() {
	leaf := s.trie.getLeaf(s.paths[0]["path"])
	s.Equal(&node{
		path: ":id",
		children: []*node{
			{
				path: "some-method-one",
			},
			{
				path: "some-method-two",
			},
		},
	}, leaf)

	leaf = s.trie.getLeaf(s.paths[1]["path"])
	s.Equal(&node{path: "some-method-one"}, leaf)

	leaf = s.trie.getLeaf(s.paths[2]["path"])
	s.Equal(&node{path: "some-method-two"}, leaf)

	leaf = s.trie.getLeaf(s.paths[3]["path"])
	s.Equal(&node{path: "ping"}, leaf)

	leaf = s.trie.getLeaf(s.paths[4]["path"])
	s.Equal(&node{path: ":id"}, leaf)
}
