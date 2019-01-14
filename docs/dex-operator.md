# Quick start:

* You must add the following flags to your API server configuration
(see [Dex's documentation](https://github.com/dexidp/dex/blob/master/Documentation/kubernetes.md#configuring-the-openid-connect-plugin)
for more details):

    ```bash
    # this should probably go in /etc/kubernetes/apiserver
    KUBE_API_ARGS="...
    --oidc-issuer-url=https://server.my-company.com:32000 \
    --oidc-client-id=kubernetes \
    --oidc-ca-file=/etc/kubernetes/pki/ca.crt \
    --oidc-username-claim=email \
    --oidc-groups-claim=groups"
    ```

    where:

    * `/etc/kubernetes/pki/ca.crt` is the CA certificate used by your API server.
    * `https://server.my-company.com:32000` is the issuer URL
    * `email` is the `nameAttr` used we will specify in the `LDAPConnector`, so Kubernetes RBAC will use it for authorizing users based on their `email`.

* Restart the API server
* Load the Dex operator

    ```
    kubectl apply -f https://raw.githubusercontent.com/kubic-project/dex-operator/master/deployments/dex-operator-full.yaml
    ```

* Once the operator is running, create a `DexConfiguration` object like this:

    ```yaml
    # my-dex-config.yaml
    apiVersion: kubic.opensuse.org/v1beta1
    kind: DexConfiguration
    metadata:
      labels:
        controller-tools.k8s.io: "1.0"
      name: dex-configuration
    spec:
      nodePort: 32000
      adminGroup: Administrators
      names:
        - server.my-company.com
        # any extra names here will be added to the certificates
    ```

    some important things:

    * the `DexConfiguration` name **must be** `dex-configuration` (otherwise it will be ignored).
    * the name and port in `https://server.my-company.com:32000` must match an entry in
    the `names` and `nodePort` attributes.

    then you can load it with `kubectl apply -f my-dex-config.yaml`.

* Add some LDAP connectors like this:

    ```yaml
    # my-connector.yaml
    apiVersion: kubic.opensuse.org/v1beta1
    kind: LDAPConnector
    metadata:
      labels:
        controller-tools.k8s.io: "1.0"
      name: external-ldap-server
    spec:
      id: some-id
      name: ldap.suse.de
      server: "ldap.suse.de:389"
      user:
        baseDn: "ou=People,dc=infra,dc=caasp,dc=local"
        filter: "(objectClass=inetOrgPerson)"
        username: mail
        idAttr: DN
        emailAttr: mail
        nameAttr: cn
        group:
      group:
        baseDn: "ou=Groups,dc=infra,dc=caasp,dc=local"
        filter: "(objectClass=groupOfUniqueNames)"
        userAttr: DN
        nameAttr: cn
        groupAttr: uniqueMember
    ```

    After loading the `LDAPConnector` with `kubectl apply -f my-connector.yaml`,
    a Dex `Deployment` should be launched automatically by the Dex operator:

    ```bash
    $ kubectl get deployment --all-namespaces                                                                                                dex_controller ✱ ◼
    NAMESPACE     NAME        DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
    kube-system   coredns     2         2         2            2           7m
    kube-system   kubic-dex   3         3         3            3           3m

    ```

    You can check the current status of the `DexConfiguration` by _"describing"_ it:

    ```bash
    $ kubectl describe dexconfiguration main-configuration                                                                                   dex_controller ✱ ◼
    Name:         main-configuration
    Namespace:
    Labels:       controller-tools.k8s.io=1.0
    Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"kubic.opensuse.org/v1beta1","kind":"DexConfiguration","metadata":{"annotations":{},"labels":{"controller-tools.k8s.io":"1.0"},"name":"ma...
    API Version:  kubic.opensuse.org/v1beta1
    Kind:         DexConfiguration
    Metadata:
      Creation Timestamp:  2018-10-03T15:09:05Z
      Finalizers:
        dexconfiguration.finalizers.kubic.opensuse.org
      Generation:        1
      Resource Version:  760
      Self Link:         /apis/kubic.opensuse.org/v1beta1/dexconfigurations/main-configuration
      UID:               44fbd63e-c71e-11e8-ae07-847beb0267d4
    Spec:
      Admin Group:  Administrators
      Certificate:
      Node Port:  32000
    Status:
      Config:      kube-system/kubic-dex
      Deployment:  kube-system/kubic-dex
      Generated Certificate:
        Name:          kubic-dex-cert
        Namespace:     kube-system
      Num Connectors:  1
      Static Passwords:
        Kubic - Dex - Cli:
          Name:       kubic-dex-cli
          Namespace:  kube-system
        Kubic - Dex - Kubernetes:
          Name:       kubic-dex-kubernetes
          Namespace:  kube-system
        Kubic - Dex - Velum:
          Name:       kubic-dex-velum
          Namespace:  kube-system
    Events:
      Type    Reason     Age   From           Message
      ----    ------     ----  ----           -------
      Normal  Checking   2m    DexController  ConfigMap 'kubic-dex' for 'main-configuration' has changed
      Normal  Checking   2m    DexController  Getting certificate 'kubic-dex-cert' for 'main-configuration'...
      Normal  Checking   2m    DexController  Deployment 'kubic-dex' for 'main-configuration' has changed
      Normal  Deploying  2m    DexController  Starting/updating Dex...
      Normal  Deploying  2m    DexController  Created 3 Secrets for shared passwords for 'main-configuration'
      Normal  Deploying  2m    DexController  Configmap 'kubic-dex' created for 'main-configuration'
      Normal  Deploying  2m    DexController  Deployment 'kubic-dex' created for 'main-configuration'
    ```

Dex will be dynamically reconfigured if you change any of these resources, so
updating the `LDAPConnector` instance or adding a new connector would result in an update
of the `ConfigMap` and a new Dex deployment, and removing all the connectors would mean that
the Dex deployment would be stopped.



# Configuration

The Dex operator can configure [Dex](https://github.com/dexidp/dex) automatically from a
set of Custom Resources. From Dex's documentation:

> Dex is an identity service that uses OpenID Connect to drive authentication for other apps.
> Dex acts as a portal to other identity providers through "connectors." This lets Dex defer
> authentication to LDAP servers, SAML providers, or established identity providers like
> GitHub, Google, and Active Directory. Clients write their authentication logic once
> to talk to Dex, then Dex handles the protocols for a given backend.
