package pgcluster

import (
	"fmt"
	"os"
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
	Suite(&ClusterErrorSuite{})
}

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

type ClusterErrorSuite struct{}

func (s *ClusterErrorSuite) TestZeroDataSourceError(c *C) {
	_, err := NewPostgreSQLCluster("postgres", []string{})
	c.Assert(err, Equals, ErrZeroDataSource)
}

func (s *ClusterErrorSuite) TestInvalidSourceError(c *C) {
	_, err := NewPostgreSQLCluster("postgresssfdf", []string{"postgres://bob/?sslmode=off"})
	c.Assert(err, Not(IsNil))
}

func (s *ClusterErrorSuite) TestDublicatedDataSource(c *C) {
	var connStrings = []string{
		"user=ubuntu dbname=circle_test",
		"user=ubuntu dbname=circle_test",
	}

	_, err := NewPostgreSQLCluster("postgres", connStrings)
	c.Assert(err, Equals, ErrDublicatedDataSource)
}

type ClusterSuite struct {
	cluster *Cluster
}

func (s *ClusterSuite) SetUpTest(c *C) {
	//docker-compose -f tests/docker-compose.yml up -d
	output, err := exec.Command("docker-compose", "-f", composeFile, "up", "-d").CombinedOutput()
	if err != nil {
		c.Fatalf("unable to start PostgreSQL cluster via compose: %v. Output %s", err, output)
	}
	// TODO: poll
	time.Sleep(10 * time.Second)

	var connStrings = getConnStrings()
	cluster, err := NewPostgreSQLCluster("postgres", connStrings)
	c.Assert(err, IsNil)

	s.cluster = cluster
}

func (s *ClusterSuite) SwitchOver(c *C) {
	if os.Getenv("CIRCLECI") == "true" {
		c.Skip("CIRCLECI uses docker with lxc-driver. Unable to switch master via pg_ctl inside a container")
	}

	// stop current master
	output, err := exec.Command("docker-compose", "-f", composeFile, "stop", "master").CombinedOutput()
	if err != nil {
		c.Fatalf("unable to stop PostgreSQL master via compose: %v. Output %s", err, output)
	}
	time.Sleep(time.Second * 1)

	// promote slave
	output, err = exec.Command("docker", "exec", "--user=postgres", "tests_slave_1",
		"/usr/lib/postgresql/9.4/bin/pg_ctl", "promote", "-D", "/var/lib/postgresql/9.4/main").CombinedOutput()
	if err != nil {
		c.Fatalf("unable to start PostgreSQL cluster via compose: %v. Output %s", err, output)
	}
	time.Sleep(time.Second * 5)
}

func (s *ClusterSuite) TearDownTest(c *C) {
	if s.cluster != nil {
		s.cluster.Close()
	}
	//docker-compose -f tests/docker-compose.yml stop
	output, err := exec.Command("docker-compose", "-f", composeFile, "stop").CombinedOutput()
	if err != nil {
		c.Fatalf("unable to stop PostgreSQL cluster via compose: %v. Output %s", err, output)
	}
}

func (s *ClusterSuite) TestNewCluster(c *C) {
	db := s.cluster.DB(MASTER)
	c.Assert(db.Ping(), IsNil)
}

func (s *ClusterSuite) TestOverWatch(c *C) {
	db1 := s.cluster.DB(MASTER)
	c.Assert(db1.Ping(), IsNil)

	time.Sleep(time.Second * 7)

	db2 := s.cluster.DB(MASTER)
	c.Assert(db2.Ping(), IsNil)

	c.Assert(db1 == db2, Equals, true)
}

func (s *ClusterSuite) TestReElect(c *C) {
	db1 := s.cluster.DB(MASTER)
	c.Assert(db1.Ping(), IsNil)

	s.SwitchOver(c)
	c.Assert(isMaster(db1), Equals, false)

	s.cluster.ReElect()
	master := s.cluster.DB(MASTER)
	c.Assert(master.Ping(), IsNil)
}
