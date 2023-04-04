# rsyslog-server
Simple docker image providing a cluster global rsyslog server

# helm install

The rsyslog-server can be installed using the helm chart under charts/


# sample Kubernetes deployment

The following deployment creates a `syslog` service to a Kubernets cluster.

Applications in the cluster may then direct syslog messages via TCP or UDP to the server `syslog`.

```yaml
kind: Deployment
apiVersion: apps/v1
metadata:
  name: syslog
  labels:
    app: syslog
spec:
  replicas: 1
  selector:
    matchLabels:
      app: syslog
  template:
    metadata:
      labels:
        app: syslog
    spec:
      containers:
      - image: kodgruvan/rsyslog-server:latest
        name: syslog
        env:
          - name: DO_LOG_ALL
            value: "false"
          - name: DO_DUMP_TO_STDOUT
            value: "false"
        ports:
          - name: syslog
            containerPort: 514
        volumeMounts:
          - name: var
            mountPath: /var/log
          - name: config
            mountPath: /etc/rsyslog.d/my-syslog.conf
            subPath: my-syslog.conf
          - name: config
            mountPath: /etc/logrotate.d/my-logrotate.conf
            subPath: my-logrotate.conf
      volumes:
        - name: var
          persistentVolumeClaim:
            claimName: syslogs
        - name: config
          configMap:
            name: my-syslog

---

kind: Service
apiVersion: v1
metadata:
  name: syslog
spec:
  selector:
    app: syslog
  ports:
    - port: 514


---

apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: syslogs
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi

---


apiVersion: v1
kind: ConfigMap
metadata:
  name: my-syslog
data:
  my-syslog.conf: |-
    *.* -/var/log/my.log
    
    # we only expect local traffic, so no point in DNS lookup of the FDQNs    
    global(net.enableDNS="off")
  my-logrotate.conf: |-
    /var/log/my.log {
        rotate 3
        copytruncate
        size 100M
        missingok
        compress
        daily
    }

```