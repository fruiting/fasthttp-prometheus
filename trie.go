package fasthttpprometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

type node struct {
	path     string
	children []*node
	metrics  map[string]prometheus.Counter
}

func (n *node) getLeaf(path string) *node {
	for i := 0; i < len(path); i++ {
		if i == len(path)-1 {
			return n.loopChildren(path, i+1, func(child *node) *node {
				return child
			})
		}

		if path[i] == slashByte && i > 0 {
			return n.loopChildren(path, i, func(child *node) *node {
				return child.getLeaf(path[i:])
			})
		}
	}

	return nil
}

func (n *node) loopChildren(path string, offset int, callback func(child *node) *node) *node {
	localPath := path[1:offset]
	for _, child := range n.children {
		if child.path == localPath || child.path[0] == colonByte {
			return callback(child)
		}
	}

	return nil
}

func (n *node) addPath(fullPath string, metricName *string) *node {
	for i := 0; i < len(fullPath); i++ {
		if i == len(fullPath)-1 {
			localPath := fullPath[1:]
			processMetricName(localPath, metricName)
			cn := &node{path: localPath}

			if len(n.children) > 0 {
				n.children = append(n.children, cn)
			} else {
				n.children = []*node{cn}
			}

			return cn
		}

		if fullPath[i] == slashByte && i > 0 {
			localPath := fullPath[1:i]
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
