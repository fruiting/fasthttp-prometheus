package fasthttpprometheus

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type trieSuite struct {
	suite.Suite

	node *node
}

func TestTrieSuite(t *testing.T) {
	suite.Run(t, &trieSuite{})
}

func (s *trieSuite) SetupTest() {
	s.node = new(node)
}

func (s *trieSuite) TestGetLeaf() {
	var metricName, metricName2, metricName3, metricName4, metricName5 string
	s.node.addPath("/ping", &metricName)
	s.node.addPath("/user/:id/action-1", &metricName2)
	s.node.addPath("/user/:id/action-2", &metricName3)
	s.node.addPath("/api/hello/", &metricName4)
	s.node.addPath("/api/hello/:name", &metricName5)

	leaf := s.node.getLeaf("")
	s.Nil(leaf)

	leaf = s.node.getLeaf("none-path/:id")
	s.Nil(leaf)

	leaf = s.node.getLeaf("/ping")
	s.Equal(&node{
		path: "/ping",
	}, leaf)

	leaf = s.node.getLeaf("/user/1/action-1")
	s.Equal(&node{
		path: "/action-1",
	}, leaf)

	leaf = s.node.getLeaf("/user/2/action-2")
	s.Equal(&node{
		path: "/action-2",
	}, leaf)

	leaf = s.node.getLeaf("/api/hello/")
	s.Equal(&node{
		path: "/hello/",
		children: []*node{
			{
				path: "/:name",
			},
		},
	}, leaf)

	leaf = s.node.getLeaf("/api/hello/test")
	s.Equal(&node{
		path: "/:name",
	}, leaf)

	leaf = s.node.getLeaf("/nil-url")
	s.Nil(leaf)
}

func (s *trieSuite) TestLoopChildren() {
	var metricName, metricName2, metricName3, metricName4, metricName5 string
	s.node.addPath("/ping", &metricName)
	s.node.addPath("/user/:id/action-1", &metricName2)
	s.node.addPath("/user/:id/action-2", &metricName3)
	s.node.addPath("/api/hello/", &metricName4)
	s.node.addPath("/api/hello/:name", &metricName5)

	child := s.node.loopChildren("", 0)
	s.Nil(child)

	child = s.node.loopChildren("/ping", 5)
	s.Equal(&node{
		path: "/ping",
	}, child)

	userNode := s.node.loopChildren("/user/1/action-1", 6)
	s.Equal(&node{
		path: "/user/",
		children: []*node{
			{
				path: "/:id/",
				children: []*node{
					{
						path: "/action-1",
					},
					{
						path: "/action-2",
					},
				},
			},
		},
	}, userNode)

	idNode := userNode.loopChildren("/1/action-1", 3)
	s.Equal(&node{
		path: "/:id/",
		children: []*node{
			{
				path: "/action-1",
			},
			{
				path: "/action-2",
			},
		},
	}, idNode)

	actionOneNode := idNode.loopChildren("/action-1", 9)
	s.Equal(&node{
		path: "/action-1",
	}, actionOneNode)

	actionTwoNode := idNode.loopChildren("/action-2", 9)
	s.Equal(&node{
		path: "/action-2",
	}, actionTwoNode)

	apiNode := s.node.loopChildren("/api/hello/", 5)
	s.Equal(&node{
		path: "/api/",
		children: []*node{
			{
				path: "/hello/",
				children: []*node{
					{
						path: "/:name",
					},
				},
			},
		},
	}, apiNode)

	helloNode := apiNode.loopChildren("/hello/", 7)
	s.Equal(&node{
		path: "/hello/",
		children: []*node{
			{
				path: "/:name",
			},
		},
	}, helloNode)

	nameVarNode := helloNode.loopChildren("/test", 5)
	s.Equal(&node{
		path: "/:name",
	}, nameVarNode)
}

func (s *trieSuite) TestAddPath() {
	var metricName, metricName2, metricName3, metricName4, metricName5, metricName6 string
	s.node.addPath("", &metricName)
	s.node.addPath("/ping", &metricName2)
	s.node.addPath("/user/:id/action-1", &metricName3)
	s.node.addPath("/user/:id/action-2", &metricName4)
	s.node.addPath("/api/hello/", &metricName5)
	s.node.addPath("/api/hello/:name", &metricName6)

	s.Equal("", metricName)
	s.Equal("ping", metricName2)
	s.Equal("user_id_var_action_1", metricName3)
	s.Equal("user_id_var_action_2", metricName4)
	s.Equal("api_hello", metricName5)
	s.Equal("api_hello_name_var", metricName6)
	s.Equal(&node{
		children: []*node{
			{
				path: "/ping",
			},
			{
				path: "/user/",
				children: []*node{
					{
						path: "/:id/",
						children: []*node{
							{
								path: "/action-1",
							},
							{
								path: "/action-2",
							},
						},
					},
				},
			},
			{
				path: "/api/",
				children: []*node{
					{
						path: "/hello/",
						children: []*node{
							{
								path: "/:name",
							},
						},
					},
				},
			},
		},
	}, s.node)
}

func (s *trieSuite) TestAppendChild() {
	metricName := "ping"
	ch := s.node.appendChild(5, "/ping", "/ping", &metricName)
	s.Equal("ping", metricName)
	s.Nil(ch)

	metricName = "user"
	ch = s.node.appendChild(5, "/user/", "/user/:id/action-1", &metricName)
	metricName = "user_id_var"
	ch = ch.appendChild(4, "/:id/", "/:id/action-1", &metricName)
	s.Equal("user_id_var_action_1", metricName)
	s.Equal(&node{
		path: "/action-1",
	}, ch)

	metricName = "user"
	ch = s.node.appendChild(5, "/user/", "/user/:id/action-2", &metricName)
	metricName = "user_id_var"
	ch = ch.appendChild(4, "/:id/", "/:id/action-2", &metricName)
	s.Equal("user_id_var_action_2", metricName)
	s.Equal(&node{
		path: "/action-2",
	}, ch)

	metricName = "api"
	ch = s.node.appendChild(4, "/api/", "/api/hello", &metricName)
	s.Equal("api_hello", metricName)
	s.Equal(&node{
		path: "/hello",
	}, ch)

	metricName = "api"
	ch = s.node.appendChild(4, "/api/", "/api/hello/:name", &metricName)
	s.Equal("api_hello_name_var", metricName)
	s.Equal(&node{
		path: "/:name",
	}, ch)
	metricName = "api_hello"
	ch = s.node.appendChild(6, "/hello/", "/hello/:name", &metricName)
	s.Equal("api_hello_name_var", metricName)
	s.Equal(&node{
		path: "/:name",
	}, ch)
}
