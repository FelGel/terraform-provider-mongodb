package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/net/proxy"
)

type ClientConfig struct {
	ConnectionString        string
	Host                    string
	Port                    string
	Username                string
	Password                string
	DB                      string
	Tls                     bool
	InsecureSkipVerify      bool
	ReplicaSet              string
	RetryWrites             bool
	Certificate             string
	Direct                  bool
	Proxy                   string
	AuthMechanism           string
	AuthMechanismProperties map[string]string
}
type DbUser struct {
	Name          string `json:"name"`
	Password      string `json:"password"`
	AuthMechanism string `json:"authMechanism,omitempty"`
}

type Role struct {
	Role string `json:"role"`
	Db   string `json:"db"`
}

func (role Role) String() string {
	return fmt.Sprintf("{ role : %s , db : %s }", role.Role, role.Db)
}

type PrivilegeDto struct {
	Db         string   `json:"db"`
	Collection string   `json:"collection"`
	Actions    []string `json:"actions"`
}

type Privilege struct {
	Resource Resource `json:"resource"`
	Actions  []string `json:"actions"`
}
type SingleResultGetUser struct {
	Users []struct {
		Id         string   `json:"_id"`
		User       string   `json:"user"`
		Db         string   `json:"db"`
		Mechanisms []string `json:"mechanisms"`
		Roles      []struct {
			Role string `json:"role"`
			Db   string `json:"db"`
		} `json:"roles"`
	} `json:"users"`
}
type SingleResultGetRole struct {
	Roles []struct {
		Role           string `json:"role"`
		Db             string `json:"db"`
		InheritedRoles []struct {
			Role string `json:"role"`
			Db   string `json:"db"`
		} `json:"inheritedRoles"`
		Privileges []struct {
			Resource struct {
				Db         string `json:"db"`
				Collection string `json:"collection"`
			} `json:"resource"`
			Actions []string `json:"actions"`
		} `json:"privileges"`
	} `json:"roles"`
}

func addArgs(arguments string, newArg string) string {
	if arguments != "" {
		return arguments + "&" + newArg
	} else {
		return "/?" + newArg
	}

}

func (c *ClientConfig) MongoClient() (*mongo.Client, error) {

	var verify = false
	var arguments = ""

	arguments = addArgs(arguments, "retrywrites="+strconv.FormatBool(c.RetryWrites))

	if c.Tls {
		arguments = addArgs(arguments, "tls=true")
	}

	if c.InsecureSkipVerify {
		verify = true
		arguments = addArgs(arguments, "tlsAllowInvalidCertificates=true")
	}

	if c.ReplicaSet != "" && !c.Direct {
		arguments = addArgs(arguments, "replicaSet="+c.ReplicaSet)
	}

	if c.Direct {
		arguments = addArgs(arguments, "connect="+"direct")
	}

	// Use connection string if given otherwise fallback to Host & Port
	uri := c.ConnectionString
	if len(uri) == 0 {
		uri = "mongodb://" + c.Host + ":" + c.Port
	}
	uri += arguments

	dialer, dialerErr := proxyDialer(c)

	if dialerErr != nil {
		return nil, dialerErr
	}

	opts := options.Client().ApplyURI(uri).SetDialer(dialer)
	if cred, ok := buildCredential(c); ok {
		opts.SetAuth(cred)
	}

	if c.Certificate != "" || verify {
		tlsConfig, err := getTLSConfig([]byte(c.Certificate), verify)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// In MongoDB driver v2, mongo.Connect() no longer accepts a context parameter
	client, err := mongo.Connect(opts)
	return client, err
}

// buildCredential assembles the driver auth credential from the client config.
// It returns ok=false when there is nothing to authenticate with (no mechanism
// and no username/password pair), leaving the connection unauthenticated.
//
// A credential is produced when an explicit auth_mechanism is set OR a
// username/password pair is present. This lets mechanisms that don't use a
// password — MONGODB-X509, MONGODB-AWS (IAM roles), MONGODB-OIDC — authenticate
// with no password. External mechanisms authenticate against the database named
// by auth_database, which must be "$external" for X.509/AWS/OIDC.
func buildCredential(c *ClientConfig) (options.Credential, bool) {
	hasUserPass := len(c.Username) > 0 && len(c.Password) > 0
	if c.AuthMechanism == "" && !hasUserPass {
		return options.Credential{}, false
	}

	cred := options.Credential{
		AuthSource:    c.DB,
		AuthMechanism: c.AuthMechanism,
		Username:      c.Username,
	}
	if len(c.AuthMechanismProperties) > 0 {
		cred.AuthMechanismProperties = c.AuthMechanismProperties
	}
	if c.Password != "" {
		cred.Password = c.Password
		cred.PasswordSet = true
	}
	return cred, true
}

func getTLSConfig(ca []byte, verify bool) (*tls.Config, error) {
	/* As of version 1.2.1, the MongoDB Go Driver will only use the first CA server certificate found in sslcertificateauthorityfile.
	   The code below addresses this limitation by manually appending all server certificates found in sslcertificateauthorityfile
	   to a custom TLS configuration used during client creation. */

	tlsConfig := new(tls.Config)

	tlsConfig.InsecureSkipVerify = verify
	if len(ca) > 0 {
		tlsConfig.RootCAs = x509.NewCertPool()
		ok := tlsConfig.RootCAs.AppendCertsFromPEM(ca)
		if !ok {
			return tlsConfig, errors.New("failed parsing pem file")
		}
	}

	return tlsConfig, nil
}

func (privilege Privilege) String() string {
	return fmt.Sprintf("{ resource : %s , actions : %s }", privilege.Resource, privilege.Actions)
}

type Resource struct {
	Db         string `json:"db"`
	Collection string `json:"collection"`
}

func (resource Resource) String() string {
	return fmt.Sprintf(" { db : %s , collection : %s }", resource.Db, resource.Collection)
}

func createIAMUser(client *mongo.Client, userName string, roles []Role, authRestrictions bson.A) error {
	rolesValue := roles
	if rolesValue == nil {
		rolesValue = []Role{}
	}
	cmd := bson.D{
		{Key: "createUser", Value: userName},
		{Key: "mechanisms", Value: bson.A{"MONGODB-AWS"}},
		{Key: "roles", Value: rolesValue},
	}
	if len(authRestrictions) > 0 {
		cmd = append(cmd, bson.E{Key: "authenticationRestrictions", Value: authRestrictions})
	}
	result := client.Database("$external").RunCommand(context.Background(), cmd)
	return result.Err()
}

func createUser(client *mongo.Client, user DbUser, roles []Role, database string, authRestrictions bson.A) error {
	var rolesValue interface{} = roles
	if len(roles) == 0 {
		rolesValue = []bson.M{}
	}
	cmd := bson.D{
		{Key: "createUser", Value: user.Name},
		{Key: "pwd", Value: user.Password},
		{Key: "roles", Value: rolesValue},
	}
	if len(authRestrictions) > 0 {
		cmd = append(cmd, bson.E{Key: "authenticationRestrictions", Value: authRestrictions})
	}
	if result := client.Database(database).RunCommand(context.Background(), cmd); result.Err() != nil {
		return result.Err()
	}
	return nil
}

func getUser(client *mongo.Client, username string, database string) (SingleResultGetUser, error) {
	result := client.Database(database).RunCommand(context.Background(), bson.D{{Key: "usersInfo", Value: bson.D{
		{Key: "user", Value: username},
		{Key: "db", Value: database},
	},
	}})
	var decodedResult SingleResultGetUser
	err := result.Decode(&decodedResult)
	if err != nil {
		return decodedResult, err
	}
	return decodedResult, nil
}

func getRole(client *mongo.Client, roleName string, database string) (SingleResultGetRole, error) {
	result := client.Database(database).RunCommand(context.Background(), bson.D{{Key: "rolesInfo", Value: bson.D{
		{Key: "role", Value: roleName},
		{Key: "db", Value: database},
	}}, {Key: "showPrivileges", Value: true}})
	var decodedResult SingleResultGetRole
	err := result.Decode(&decodedResult)
	if err != nil {
		return decodedResult, err
	}
	return decodedResult, nil
}

func createRole(client *mongo.Client, role string, roles []Role, privilege []PrivilegeDto, database string, authRestrictions bson.A) error {
	var privileges []Privilege
	for _, element := range privilege {
		var prv Privilege
		prv.Resource = Resource{
			Db:         element.Db,
			Collection: element.Collection,
		}
		prv.Actions = element.Actions
		privileges = append(privileges, prv)
	}

	var privilegesValue interface{} = privileges
	if len(privileges) == 0 {
		privilegesValue = []bson.M{}
	}
	var rolesValue interface{} = roles
	if len(roles) == 0 {
		rolesValue = []bson.M{}
	}
	cmd := bson.D{
		{Key: "createRole", Value: role},
		{Key: "privileges", Value: privilegesValue},
		{Key: "roles", Value: rolesValue},
	}
	if len(authRestrictions) > 0 {
		cmd = append(cmd, bson.E{Key: "authenticationRestrictions", Value: authRestrictions})
	}

	if result := client.Database(database).RunCommand(context.Background(), cmd); result.Err() != nil {
		return result.Err()
	}
	return nil
}

func MongoClientInit(conf *MongoDatabaseConfiguration) (*mongo.Client, error) {

	client, err := conf.Config.MongoClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), conf.MaxConnLifetime*time.Second)
	defer cancel()
	// client.Connect is deprecated, already connected by mongo.Connect above
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func proxyDialer(c *ClientConfig) (options.ContextDialer, error) {
	proxyFromEnv := proxy.FromEnvironment().(options.ContextDialer)
	proxyFromProvider := c.Proxy

	if len(proxyFromProvider) > 0 {
		proxyURL, err := url.Parse(proxyFromProvider)
		if err != nil {
			return nil, err
		}
		proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return proxyDialer.(options.ContextDialer), nil
	}

	return proxyFromEnv, nil
}
