# Changelog

## 3.1.0

FEATURES:

* `mongodb_db_user` and `mongodb_db_role`: `authentication_restriction` blocks (`client_source` / `server_address`), mapping to MongoDB's `authenticationRestrictions`. Restricts the IP addresses/CIDR ranges a user (or a user granted the role) may connect from and to. Available in Community MongoDB 3.6+.
* **provider**: `auth_mechanism` and `auth_mechanism_properties` configure the SASL mechanism the provider uses for its own connection — unlocking `MONGODB-X509`, `MONGODB-AWS`, and `MONGODB-OIDC` (plus explicit `SCRAM-SHA-1`/`SCRAM-SHA-256`) in addition to the previously hard-coded SCRAM path. `auth_mechanism` can also be sourced from the `MONGO_AUTH_MECHANISM` environment variable.
* `mongodb_db_index`: `sparse` indexes, via a `keys` entry with `field = "sparse"` (mirroring the existing `unique` / `expireAfterSeconds` entries).

NOTES:

* All changes are additive and backward-compatible; existing configurations and state are unaffected.

## 3.0.0

BREAKING CHANGES:

* The provider now serves **Terraform Plugin Protocol 6** and requires **Terraform 1.0 or later** (previously protocol 5 / Terraform 0.12+). It was re-architected from `terraform-plugin-sdk/v2` to `terraform-plugin-framework`, served through `terraform-plugin-mux`. The migration is behavior-preserving and existing state upgrades cleanly (verified against 2.0.4).
* `mongodb_db_user`: `password` is no longer `Required` — it is now `Optional` (a password is still required for standard users unless `password_wo` is used). `auth_database` is now `Optional` (defaults to `$external` for `MONGODB-AWS` users).

FEATURES:

* `mongodb_db_user`: IAM authentication for Amazon DocumentDB via `auth_mechanism = "MONGODB-AWS"` (users are created in `$external`; `name` must be an AWS IAM ARN).
* `mongodb_db_user`: write-only password via `password_wo` + `password_wo_version` — the password is never stored in Terraform state (requires Terraform 1.11+).
* `mongodb_db_index`: `hidden` indexes (toggleable via `collMod`) and `partial_filter_expression` for partial indexes.
* **List resources** for `mongodb_db_user`, `mongodb_db_role`, `mongodb_db_collection`, and `mongodb_db_index` — enumerate existing objects with `terraform query` (requires Terraform 1.14+).

NOTES:

* All four resources now expose a resource identity (`id`).
