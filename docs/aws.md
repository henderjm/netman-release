## Deploy to AWS
0. Upload stemcell with Linux kernel 4.4 to bosh director.  Versions >= 3263.2 should work.

0. Generate credentials
  - Create a strong password for a new UAA client to be called `network-policy`.  We'll refer to this
    with the string `REPLACE_WITH_UAA_CLIENT_SECRET` below.
  - Generate certs & keys for mutual TLS between the policy server and policy agents.  You can use our
    [handy script](../scripts/generate-certs) to create these.  We'll refer to these with the strings

    ```
    REPLACE_WITH_CA_CERT
    REPLACE_WITH_CLIENT_CERT
    REPLACE_WITH_CLIENT_KEY
    REPLACE_WITH_SERVER_CERT
    REPLACE_WITH_SERVER_KEY
    ```

0. Edit the CF properties stub

  - Add under `properties.uaa.scim.users` the group `network.admin` for `admin`
    ```diff
    scim:
      users:
      - name: admin
        password: <admin-password>
        groups:
          - scim.write
          - scim.read
          - openid
          - cloud_controller.admin
          - clients.read
          - clients.write
          - doppler.firehose
          - routing.router_groups.read
          - routing.router_groups.write
    +     - network.admin
    ```

  - Add under `properties.uaa.clients`

    ```diff
    clients:
      cf:
    -   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read
    +   scope: cloud_controller.read,cloud_controller.write,openid,password.write,cloud_controller.admin,scim.read,scim.write,doppler.firehose,uaa.user,routing.router_groups.read,network.admin
    + network-policy:
    +   authorities: uaa.resource,cloud_controller.admin_read_only
    +   authorized-grant-types: client_credentials,refresh_token
    +   secret: REPLACE_WITH_UAA_CLIENT_SECRET
    ```


0. Create a netman stub `stubs/netman/stub.yml`:

    ```yaml
    ---
    netman_overrides:
      releases:
      - name: netman
        version: latest
      driver_templates:
      - name: garden-cni
        release: netman
      - name: cni-flannel
        release: netman
      - name: netmon
        release: netman
      - name: vxlan-policy-agent
        release: netman
      properties:
        vxlan-policy-agent:
          policy_server_url: https://policy-server.service.cf.internal:4003
          ca_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_CA_CERT
            -----END CERTIFICATE-----
          client_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_CLIENT_CERT
            -----END CERTIFICATE-----
          client_key: |
            -----BEGIN RSA PRIVATE KEY-----
            REPLACE_WITH_CLIENT_KEY
            -----END RSA PRIVATE KEY-----
        policy-server:
          uaa_client_secret: REPLACE_WITH_UAA_CLIENT_SECRET
          uaa_url: (( "https://uaa." config_from_cf.system_domain ))
          cc_url: (( "https://api." config_from_cf.system_domain ))
          skip_ssl_validation: true
          database:
            # For MySQL use these two lines
            type: mysql
            connection_string: USERNAME:PASSWORD@tcp(DB_HOSTNAME:3306)/DB_NAME
            # OR for Postgres, use these
            # type: postgres
            # connection_string: postgres://USERNAME:PASSWORD@DB_HOSTNAME:5524/DB_NAME?sslmode=disable
          ca_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_CA_CERT
            -----END CERTIFICATE-----
          server_cert: |
            -----BEGIN CERTIFICATE-----
            REPLACE_WITH_SERVER_CERT
            -----END CERTIFICATE-----
          server_key: |
            -----BEGIN RSA PRIVATE KEY-----
            REPLACE_WITH_SERVER_KEY
            -----END RSA PRIVATE KEY-----
        garden-cni:
          cni_plugin_dir: /var/vcap/packages/flannel/bin
          cni_config_dir: /var/vcap/jobs/cni-flannel/config/cni
        cni-flannel:
          flannel:
            etcd:
              require_ssl: (( config_from_cf.etcd.require_ssl))
          etcd_endpoints:
            - (( config_from_cf.etcd.advertise_urls_dns_suffix ))
          etcd_client_cert: (( config_from_cf.etcd.client_cert ))
          etcd_client_key: (( config_from_cf.etcd.client_key ))
          etcd_ca_cert: (( config_from_cf.etcd.ca_cert ))
      garden_properties:
        network_plugin: /var/vcap/packages/runc-cni/bin/garden-external-networker
        network_plugin_extra_args:
        - --configFile=/var/vcap/jobs/garden-cni/config/adapter.json
      jobs:
      - name: policy-server
        instances: 1
        persistent_disk: 256
        templates:
        - name: policy-server
          release: netman
        - name: route_registrar
          release: cf
        - name: consul_agent
          release: cf
        - name: metron_agent
          release: cf
        resource_pool: database_z1
        networks:
          - name: diego1
        properties:
          nats:
            machines: (( config_from_cf.nats.machines ))
            user: (( config_from_cf.nats.user ))
            password: (( config_from_cf.nats.password ))
            port: (( config_from_cf.nats.port ))
          metron_agent:
            zone: z1
          route_registrar:
            routes:
            - name: policy-server
              port: 4002
              registration_interval: 20s
              uris:
              - (( "api." config_from_cf.system_domain "/networking" ))
          consul:
            agent:
              services:
                policy-server:
                  name: policy-server
                  check:
                    interval: 5s
                    script: /bin/true

    config_from_cf: (( merge ))
    ```

0. Generate diego with netman manifest:
  - Run the following bash script. Set `environment_path` to the directory containing your stubs for cf, diego, and netman.
    Set `output_path` to the directory you want your manifest to be created in.
    Set `diego_release_path` to your local copy of the diego-release repository.

  ```bash
  set -e -x -u

  environment_path=
  output_path=
  diego_release_path=

  pushd cf-release
    ./scripts/generate_deployment_manifest aws \
      ${environment_path}/stubs/director-uuid.yml \
      ${diego_release_path}/examples/aws/stubs/cf/diego.yml \
      ${environment_path}/stubs/cf/properties.yml \
      ${environment_path}/stubs/cf/instance-count-overrides.yml \
      ${environment_path}/stubs/cf/stub.yml \
      > ${output_path}/cf.yml
  popd

  pushd diego-release
    ./scripts/generate-deployment-manifest \
      -g \
      -c ${output_path}/cf.yml \
      -i ${environment_path}/stubs/diego/iaas-settings.yml \
      -p ${environment_path}/stubs/diego/property-overrides.yml \
      -n ${environment_path}/stubs/diego/instance-count-overrides.yml \
      -N ${environment_path}/stubs/netman/stub.yml \
      -v ${environment_path}/stubs/diego/release-versions.yml \
      > ${output_path}/diego.yml
  popd
  ```

0. Deploy
  - Target your bosh director.
  ```bash
  bosh target <your-director>
  ```
  - Set the deployment
  ```bash
  bosh deployment ${output_path}/diego.yml
  ```
  - Deploy
  ```bash
  bosh deploy
  ```


0. Kicking the tires

   Try out our [Cats and Dogs example](../src/example-apps/cats-and-dogs) on your new deployment.
