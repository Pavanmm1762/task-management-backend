package utils

// utils/cassandra.go

import (
	"fmt"
	"log"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

var Session *gocql.Session

func InitCassandra() {
	// Create a new Cassandra cluster
	cluster := gocql.NewCluster("d49c0770-dd1b-4a0b-bde9-e300d929a942-eu-west-1.db.astra.datastax.com") // Astra DB Endpoint
	cluster.Port = 29042
	cluster.Keyspace = "task_management_team_collaboration"

	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: "kDlgtZqvFgHfNvgJoqWOUPpr",
		Password: "-dyaJZZ1uS1PCdRz0,k.zhfC_nq0krHW1upBDftgA2K8D7RloZ.oj3uJoaoDFzoWBmIFTLqcH9Q05idEu_E95a5JT1Oa5aGgNfMOg_ZDXFImuD06sRYvyCo4mpupDJRG",
	}
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4

	cluster.SslOpts = &gocql.SslOptions{
		CaPath:                 "./secure-connect-pavan/ca.crt",
		CertPath:               "./secure-connect-pavan/cert",
		KeyPath:                "./secure-connect-pavan/key",
		EnableHostVerification: false,
	}

	// Create session
	var err error
	Session, err = cluster.CreateSession()
	if err != nil {
		log.Fatalf("Failed to connect to Astra DB: %v", err)
		panic(err)
	}
	fmt.Println("Successfully connected to Astra DB")

	// Test connection
	fmt.Println("Running a test query to check Cassandra version...")

	var releaseVersion string
	if err := Session.Query("SELECT release_version FROM system.local").Scan(&releaseVersion); err != nil {
		log.Fatalf("Failed to run version query: %v", err)
	} else {
		fmt.Printf("Cassandra version: %s\n", releaseVersion)
	}

	fmt.Println("Attempting to create the admin table...")

}

func GenerateUUID() string {
	return uuid.New().String()
}
