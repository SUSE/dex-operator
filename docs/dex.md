# Configuration

The Dex operator can configure [Dex](https://github.com/dexidp/dex) automatically from a
set of Custom Resources. From Dex's documentation:

> Dex is an identity service that uses OpenID Connect to drive authentication for other apps.
> Dex acts as a portal to other identity providers through "connectors." This lets Dex defer
authentication to LDAP servers, SAML providers, or established identity providers like
GitHub, Google, and Active Directory. Clients write their authentication logic once
to talk to Dex, then Dex handles the protocols for a given backend.
