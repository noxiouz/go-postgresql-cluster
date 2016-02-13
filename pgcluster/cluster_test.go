package pgcluster

import (
	"testing"
	"time"

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

func (s *ClusterSuite) TestZeroDataSourceError(c *C) {
	_, err := NewPostgreSQLCluster("postgres", []string{})
	c.Assert(err, Equals, ErrZeroDataSource)
}

func (s *ClusterSuite) TestDublicatedDataSource(c *C) {
	var connStrings = []string{
		"user=ubuntu dbname=circle_test",
		"user=ubuntu dbname=circle_test",
	}

	_, err := NewPostgreSQLCluster("postgres", connStrings)
	c.Assert(err, Equals, ErrDublicatedDataSource)
}

func (s *ClusterSuite) TestOverWatch(c *C) {
	var connStrings = []string{
		"user=ubuntu dbname=circle_test",
		"user=user1 dbname=testdb",
	}

	cluster, err := NewPostgreSQLCluster("postgres", connStrings)
	c.Assert(err, IsNil)
	defer cluster.Close()

	db1 := cluster.DB(MASTER)
	err = db1.Ping()
	c.Assert(err, IsNil)

	time.Sleep(time.Second * 10)
	db2 := cluster.DB(MASTER)
	err = db2.Ping()
	c.Assert(err, IsNil)

	c.Assert(db1 == db2, Equals, true)
}
