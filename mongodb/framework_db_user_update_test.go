package mongodb

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAppendAuthRestrictionsOnUpdate(t *testing.T) {
	t.Parallel()

	base := bson.D{{Key: "updateUser", Value: "u"}, {Key: "roles", Value: []Role{}}}
	set := bson.A{bson.D{{Key: "clientSource", Value: []string{"127.0.0.1/32"}}}}

	t.Run("omit when never configured", func(t *testing.T) {
		got := appendAuthRestrictionsOnUpdate(base, nil, nil)
		if _, ok := lookupBSON(got, "authenticationRestrictions"); ok {
			t.Fatalf("expected field omitted, got %#v", got)
		}
	})

	t.Run("send plan value when set", func(t *testing.T) {
		got := appendAuthRestrictionsOnUpdate(base, set, nil)
		v, ok := lookupBSON(got, "authenticationRestrictions")
		if !ok {
			t.Fatal("expected field present")
		}
		if len(v.(bson.A)) != 1 {
			t.Fatalf("expected plan restrictions, got %#v", v)
		}
	})

	t.Run("send empty array to clear", func(t *testing.T) {
		got := appendAuthRestrictionsOnUpdate(base, nil, set)
		v, ok := lookupBSON(got, "authenticationRestrictions")
		if !ok {
			t.Fatal("expected field present to clear")
		}
		arr, ok := v.(bson.A)
		if !ok || len(arr) != 0 {
			t.Fatalf("expected empty array, got %#v", v)
		}
	})
}

func lookupBSON(cmd bson.D, key string) (any, bool) {
	for _, e := range cmd {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}
