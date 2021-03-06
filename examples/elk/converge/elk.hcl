param "user-name" {
  default = "vagrant"
}

module "packages.hcl" "packages" {}

module "docker.hcl" "docker" {
  params = {
    user-name = "{{param `user-name`}}"
  }

  depends = ["module.packages"]
}

param "elasticsearch-data-directory" {
  default = "/data/elasticsearch"
}

param "filebeat-service" {
  default = "filebeat"
}

task "filebeat-install" {
  check   = "yum list installed filebeat"
  apply   = "rpm -ivh https://download.elastic.co/beats/filebeat/filebeat-1.3.0-x86_64.rpm"
  depends = ["module.docker"]
}

file.content "filebeat-yml" {
  destination = "/etc/filebeat/filebeat.yml"

  content = <<EOF
filebeat:
  prospectors:
    - paths:
        - /var/log/*.log
        - /var/log/messages
      input_type: log
  registry_file: /var/lib/filebeat/registry
output:
  elasticsearch:
    hosts: ["localhost:9200"]
EOF

  depends = ["task.filebeat-install"]
}

task "filebeat-enable" {
  check   = "systemctl is-enabled {{param `filebeat-service`}}"
  apply   = "systemctl enable {{param `filebeat-service`}}"
  depends = ["file.content.filebeat-yml"]
}

wait.query "elasticsearch-wait" {
  check = <<EOF
status=$(curl -s 'http://localhost:9200/_cluster/health' 2>/dev/null | jq -r .status)
[[ "$status" == "yellow" ]] || [[ "$status" == "green" ]]
EOF

  interval  = "10s"
  max_retry = 10

  depends = ["docker.container.elasticsearch-container"]
}

task "filebeat-elasticsearch-template" {
  check   = "[[  \"$(curl 'http://localhost:9200/_template/filebeat' 2>/dev/null)\" != \"{}\" ]] || exit 1"
  apply   = "curl -XPUT 'http://localhost:9200/_template/filebeat' -d@/etc/filebeat/filebeat.template.json 2>/dev/null"
  depends = ["task.filebeat-enable", "docker.container.elasticsearch-container", "wait.query.elasticsearch-wait"]
}

task "filebeat-start" {
  check   = "systemctl is-active {{param `filebeat-service`}}"
  apply   = "systemctl start {{param `filebeat-service`}}"
  depends = ["task.filebeat-enable", "docker.container.elasticsearch-container"]
}

file.directory "elasticsearch-data-directory" {
  destination = "{{param `elasticsearch-data-directory`}}"
  create_all  = true
}

docker.image "elasticsearch-image" {
  name    = "elasticsearch"
  tag     = "2.4.0"
  depends = ["module.docker"]
}

docker.container "elasticsearch-container" {
  name    = "elasticsearch"
  image   = "{{lookup `docker.image.elasticsearch-image.name`}}:{{lookup `docker.image.elasticsearch-image.tag`}}"
  command = ["elasticsearch", "-Des.insecure.allow.root=true"]
  ports   = ["127.0.0.1:9200:9200"]
  volumes = ["{{param `elasticsearch-data-directory`}}:/usr/share/elasticsearch/data"]
  force   = "true"
  depends = ["file.directory.elasticsearch-data-directory"]
}

docker.image "kibana-image" {
  name    = "kibana"
  tag     = "4.6.0"
  depends = ["module.docker"]
}

docker.container "kibana-container" {
  name  = "kibana"
  image = "{{lookup `docker.image.kibana-image.name`}}:{{lookup `docker.image.kibana-image.tag`}}"
  ports = ["5601:5601"]
  links = ["{{lookup `docker.container.elasticsearch-container.name`}}:elasticsearch"]
  force = "true"
}
