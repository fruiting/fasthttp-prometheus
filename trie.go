package fasthttpprometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// prefix tree node
// every node contains route part and may contain child nodes and metrics
// for example you have two routes:
//
// /user/:id/action-1
// /user/:id/action-2
//
// it will make next tree:
//
// _________user__________
// ___________|___________
// __________:id__________
// _________/___\_________
// _action-1_____action_2_
// _metrics_______metrics_
//
// leaf with part = action contains total and failure_total metrics for full route
type node struct {
	path     string
	children []*node
	metrics  map[string]prometheus.Counter
}

// getLeaf returns leaf with metrics for full route
func (n *node) getLeaf(path string) *node {
	for i := 0; i < len(path); i++ {
		if i == len(path)-1 {
			return n.loopChildren(path, i+1)
		}

		if path[i] == slashByte && i > 0 {
			if cn := n.loopChildren(path, i+1); cn != nil {
				return cn.getLeaf(path[i:])
			}

			return nil
		}
	}

	return nil
}

func (n *node) loopChildren(path string, offset int) *node {
	localPath := path[:offset]
	for _, child := range n.children {
		if child.path == localPath {
			return child
		}
		if child.path[1] == colonByte {
			if child.path[len(child.path)-1] == slashByte && localPath[len(localPath)-1] == slashByte {
				return child
			} else if child.path[len(child.path)-1] != slashByte && localPath[len(localPath)-1] != slashByte {
				return child
			}
		}
	}

	return nil
}

// addPath splits route and inserts new nodes in the trie
func (n *node) addPath(fullPath string, metricName *string) *node {
	for i := 0; i < len(fullPath); i++ {
		if i == len(fullPath)-1 {
			processMetricName(fullPath, metricName)
			cn := &node{path: fullPath}

			if len(n.children) > 0 {
				n.children = append(n.children, cn)
			} else {
				n.children = []*node{cn}
			}

			return cn
		}

		if fullPath[i] == slashByte && i > 0 {
			localPath := fullPath[:i+1]
			processMetricName(localPath, metricName)

			if len(n.children) > 0 {
				for _, child := range n.children {
					if child.path == localPath || child.path[0] == colonByte {
						return child.addPath(fullPath[i:], metricName)
					}
				}

				return n.appendChild(i, localPath, fullPath, metricName)
			} else {
				return n.appendChild(i, localPath, fullPath, metricName)
			}
		}
	}

	return nil
}

func (n *node) appendChild(offset int, localPath string, fullPath string, metricName *string) *node {
	child := &node{path: localPath}
	n.children = append(n.children, child)

	return child.addPath(fullPath[offset:], metricName)
}
