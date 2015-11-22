When(/^I start a new application with one task specifying all container options$/) do
  response = create_application(
    goals: {
      task1: {
        image: "alpine:3.2",
        command: ["echo executed"],
        task: true,
        extra_hosts: ["googledns:8.8.8.8"],
        ports: ["3000", "49100:22", "33/udp", "53:53/udp"],
        expose: ["3000","2000/udp"],
        volumes: ["/tmp", "/tmp:/foo", "/tmp:/bar:ro"],
        environment: {
          "FOO": "BAR",
          "BUZ": "LIGHTYEAR"
        },
        labels: {
          "foo": "bar"
        },
        log_driver: "syslog",
        log_config: {
          "syslog-address": "udp://192.168.0.42:123"
        },
        net: "bridge",
        dns: ["1.2.3.4"],
        cap_add: ['NET_ADMIN'],
        cap_drop: ['SYS_ADMIN'],
        dns_search: ['netice9.com'],
        devices: ["/dev/tty", "/dev/console:/dev/con", "/dev/ttyS0:mr", "/dev/ttyS1:/dev/XT:r"],
        security_opt: ['label:role:ROLE'],
        working_dir: '/tmp',
        entrypoint: ['/bin/sh','-c'],
        user: "root",
        hostname: "test-host",
        domainname: "netice9.com",
        mac_address: "02:42:ac:11:05:0e",
        mem_limit: 8_000_000,
        memswap_limit: 16_000_000,
        privileged: true,
        restart: "on-failure",
        stdin_open: true,
        tty: false,
        cpu_shares: 73,
        cpuset: "0",
        read_only: true,
        volume_driver: "local"

      }
    },
    main_goal: 'task1'
  )
end

Then(/^ExtraHosts should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['ExtraHosts']).to eq(["googledns:8.8.8.8"])
end

Then(/^PortBindings should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['PortBindings']).to eq({"22/tcp"=>[{"HostPort"=>"49100"}], "3000/tcp"=>[], "33/udp"=>[], "53/udp"=>[{"HostPort"=>"53"}]})
end

Then(/^ExposedPorts should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['ExposedPorts']).to eq({"3000/tcp"=>{}, "2000/udp"=>{}})
end

Then(/^Binds should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['Binds']).to eq(["/tmp:/tmp", "/tmp:/foo", "/tmp:/bar:ro"])
end

Then(/^Env should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['Env'].sort).to eq(["FOO=BAR", "BUZ=LIGHTYEAR"].sort)
end

Then(/^Labels should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['Labels']).to eq({"foo"=>"bar"})
end

Then(/^LogConfig should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['LogConfig']).to eq({"Type"=>"syslog", "Config"=>{"syslog-address"=>"udp://192.168.0.42:123"}})
end

Then(/^NetworkMode should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['NetworkMode']).to eq("bridge")
end

Then(/^Dns should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['Dns']).to eq(["1.2.3.4"])
end

Then(/^CapAdd should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['CapAdd']).to eq(["NET_ADMIN"])
end

Then(/^CapDrop should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['CapDrop']).to eq(["SYS_ADMIN"])
end

Then(/^DnsSearch should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['DnsSearch']).to eq(["netice9.com"])
end

Then(/^Devices should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['Devices']).to eq(
    [
      {"PathOnHost"=>"/dev/tty", "PathInContainer"=>"/dev/tty", "CgroupPermissions"=>"mrw"},
      {"PathOnHost"=>"/dev/console", "PathInContainer"=>"/dev/con", "CgroupPermissions"=>"mrw"},
      {"PathOnHost"=>"/dev/ttyS0", "PathInContainer"=>"/dev/ttyS0", "CgroupPermissions"=>"mr"},
      {"PathOnHost"=>"/dev/ttyS1", "PathInContainer"=>"/dev/XT", "CgroupPermissions"=>"r"}
    ]
  )
end

Then(/^SecurityOpt should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['SecurityOpt']).to eq(['label:role:ROLE'])
end

Then(/^WorkingDir should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['WorkingDir']).to eq('/tmp')
end

Then(/^Entrypoint should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['Entrypoint']).to eq(['/bin/sh', '-c'])
end

Then(/^User should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['User']).to eq("root")
end

Then(/^Hostname should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['Hostname']).to eq("test-host")
end

Then(/^Domainname should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['Domainname']).to eq("netice9.com")
end

Then(/^MacAddress should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['MacAddress']).to eq("02:42:ac:11:05:0e")
end

Then(/^Memory should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['Memory']).to eq(8_000_000)
end

Then(/^MemorySwap should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['MemorySwap']).to eq(16_000_000)
end

Then(/^Privileged should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['Privileged']).to eq(true)
end

Then(/^RestartPolicy should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['RestartPolicy']).to eq({"Name"=>"on-failure"})
end

Then(/^AttachStdin should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['AttachStdin']).to eq(true)
end

Then(/^Tty should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['Tty']).to eq(true)
end


Then(/^CpuShares should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['CpuShares']).to eq(73)
end

Then(/^CpusetCpus should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['CpusetCpus']).to eq("0")
end

Then(/^CpusetMems should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['CpusetMems']).to eq("0")
end

Then(/^ReadonlyRootfs should be set$/) do
  expect(inspect_goal('task1').to_h['HostConfig']['ReadonlyRootfs']).to eq(true)
end

Then(/^VolumeDriver should be set$/) do
  expect(inspect_goal('task1').to_h['Config']['VolumeDriver']).to eq("local")
end

