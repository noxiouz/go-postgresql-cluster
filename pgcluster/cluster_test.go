package pgcluster

import (
	"fmt"
	"os/exec"
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

const (
	composeFile = "../tests/docker-compose.yml"
)

// Look at tests/docker-compose.yml
func getConnStrings() []string {
	var (
		dockerMachineName = "dev"
		host              = ""
	)

	// MacOS X: detect docker-machine IP
	if output, err := exec.Command("docker-machine", "ip", dockerMachineName).Output(); err == nil {
		host = fmt.Sprintf(" host=%s", output)
	}

	// look at tests/docker-compose.yml
	return []string{
		// slave
		"user=dbuser dbname=dbname password=dbuserpass sslmode=disable port=7432" + host,
		// master
		"user=dbuser dbname=dbname password=dbuserpass sslmode=disable port=6432" + host,
	}
}

func (s *ClusterSuite) SetUpSuite(c *C) {
	//docker-compose -f tests/docker-compose.yml up -d
	output, err := exec.Command("docker-compose", "-f", composeFile, "up", "-d", "--force-recreate").CombinedOutput()
	if err != nil {
		c.Fatalf("unable to start PostgreSQL cluster via compose: %v. Output %s", err, output)
	}
	// TODO: poll
	time.Sleep(10 * time.Second)
}

func (s *ClusterSuite) TearDownSuite(c *C) {
	//docker-compose -f tests/docker-compose.yml stop
	output, err := exec.Command("docker-compose", "-f", composeFile, "stop").CombinedOutput()
	if err != nil {
		c.Fatalf("unable to stop PostgreSQL cluster via compose: %v. Output %s", err, output)
	}
}

func (s *ClusterSuite) TestNewCluster(c *C) {
	var connStrings = getConnStrings()
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
	var connStrings = getConnStrings()

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
