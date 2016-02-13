package pgcluster

import (
	"testing"

	_ "github.com/lib/pq"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

func init() {
	Suite(&ClusterSuite{})
}

type ClusterSuite struct{}

func (s *ClusterSuite) TestNewCluster(c *C) {
	var connStrings = []string{
		"user=ubuntu dbname=circle_test",
		"user=user1 dbname=testdb",
	}
	cluster, err := NewPostgreSQLCluster("postgres", connStrings)
	c.Assert(err, IsNil)
	defer cluster.Close()

	db := cluster.DB(MASTER)
	err = db.Ping()
	c.Assert(err, IsNil)
}
