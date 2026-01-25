//go:build integration

// Integration tests for LDAP authentication
// Run with: go test -tags=integration ./tests/integration/...
//
// Prerequisites:
// - docker compose -f docker-compose.test.yaml up -d openldap
// - Wait for LDAP server to be healthy

package integration

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/go-ldap/ldap/v3"
)

func TestLDAP_Connection(t *testing.T) {
	host := os.Getenv("LDAP_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("LDAP_PORT")
	if port == "" {
		port = "389"
	}

	conn, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		t.Skipf("Skipping: LDAP not available: %v", err)
	}
	defer conn.Close()

	t.Log("LDAP connection successful")
}

func TestLDAP_AdminBind(t *testing.T) {
	host := os.Getenv("LDAP_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("LDAP_PORT")
	if port == "" {
		port = "389"
	}
	adminDN := os.Getenv("LDAP_ADMIN_DN")
	if adminDN == "" {
		adminDN = "cn=admin,dc=k13d,dc=test"
	}
	adminPassword := os.Getenv("LDAP_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "adminpassword"
	}

	conn, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		t.Skipf("Skipping: LDAP not available: %v", err)
	}
	defer conn.Close()

	// Bind as admin
	err = conn.Bind(adminDN, adminPassword)
	if err != nil {
		t.Fatalf("Failed to bind as admin: %v", err)
	}

	t.Log("LDAP admin bind successful")
}

func TestLDAP_Search(t *testing.T) {
	host := os.Getenv("LDAP_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("LDAP_PORT")
	if port == "" {
		port = "389"
	}
	baseDN := os.Getenv("LDAP_BASE_DN")
	if baseDN == "" {
		baseDN = "dc=k13d,dc=test"
	}
	adminDN := os.Getenv("LDAP_ADMIN_DN")
	if adminDN == "" {
		adminDN = "cn=admin,dc=k13d,dc=test"
	}
	adminPassword := os.Getenv("LDAP_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "adminpassword"
	}

	conn, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		t.Skipf("Skipping: LDAP not available: %v", err)
	}
	defer conn.Close()

	// Bind as admin
	err = conn.Bind(adminDN, adminPassword)
	if err != nil {
		t.Skipf("Skipping: cannot bind: %v", err)
	}

	// Search for all entries
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"dn", "cn"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		t.Fatalf("LDAP search failed: %v", err)
	}

	t.Logf("Found %d entries", len(result.Entries))
	for _, entry := range result.Entries {
		t.Logf("  DN: %s", entry.DN)
	}
}

func TestLDAP_CreateUser(t *testing.T) {
	host := os.Getenv("LDAP_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("LDAP_PORT")
	if port == "" {
		port = "389"
	}
	baseDN := os.Getenv("LDAP_BASE_DN")
	if baseDN == "" {
		baseDN = "dc=k13d,dc=test"
	}
	adminDN := os.Getenv("LDAP_ADMIN_DN")
	if adminDN == "" {
		adminDN = "cn=admin,dc=k13d,dc=test"
	}
	adminPassword := os.Getenv("LDAP_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "adminpassword"
	}

	conn, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		t.Skipf("Skipping: LDAP not available: %v", err)
	}
	defer conn.Close()

	// Bind as admin
	err = conn.Bind(adminDN, adminPassword)
	if err != nil {
		t.Skipf("Skipping: cannot bind: %v", err)
	}

	// Create OU for users first (if it doesn't exist)
	ouRequest := ldap.NewAddRequest("ou=users,"+baseDN, nil)
	ouRequest.Attribute("objectClass", []string{"organizationalUnit"})
	ouRequest.Attribute("ou", []string{"users"})
	_ = conn.Add(ouRequest) // Ignore error if exists

	// Create test user
	userDN := "cn=testuser,ou=users," + baseDN
	addRequest := ldap.NewAddRequest(userDN, nil)
	addRequest.Attribute("objectClass", []string{"inetOrgPerson"})
	addRequest.Attribute("cn", []string{"testuser"})
	addRequest.Attribute("sn", []string{"User"})
	addRequest.Attribute("givenName", []string{"Test"})
	addRequest.Attribute("mail", []string{"testuser@k13d.test"})
	addRequest.Attribute("userPassword", []string{"testpassword"})

	err = conn.Add(addRequest)
	if err != nil {
		// User might already exist
		if !ldap.IsErrorWithCode(err, ldap.LDAPResultEntryAlreadyExists) {
			t.Fatalf("Failed to create user: %v", err)
		}
		t.Log("Test user already exists")
	} else {
		t.Log("Test user created successfully")
	}

	// Verify user can authenticate
	userConn, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		t.Fatalf("Failed to create user connection: %v", err)
	}
	defer userConn.Close()

	err = userConn.Bind(userDN, "testpassword")
	if err != nil {
		t.Fatalf("User authentication failed: %v", err)
	}

	t.Log("Test user authentication successful")

	// Clean up - delete test user
	delRequest := ldap.NewDelRequest(userDN, nil)
	err = conn.Del(delRequest)
	if err != nil {
		t.Logf("Warning: failed to delete test user: %v", err)
	}
}

func TestLDAP_StartTLS(t *testing.T) {
	host := os.Getenv("LDAP_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("LDAP_PORT")
	if port == "" {
		port = "389"
	}

	conn, err := ldap.DialURL("ldap://" + host + ":" + port)
	if err != nil {
		t.Skipf("Skipping: LDAP not available: %v", err)
	}
	defer conn.Close()

	// Attempt StartTLS (may fail in test environment without proper certs)
	err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Logf("StartTLS not available (expected in test environment): %v", err)
		return
	}

	t.Log("StartTLS upgrade successful")
}
