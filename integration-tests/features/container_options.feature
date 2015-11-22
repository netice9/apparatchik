Feature: container options

Scenario: set all container options
  When I start a new application with one task specifying all container options
  When I wait for the task to finish
  Then ExtraHosts should be set
  And PortBindings should be set
  And ExposedPorts should be set
  And Binds should be set
  And Env should be set
  And Labels should be set
  And LogConfig should be set
  And NetworkMode should be set
  And Dns should be set
  And CapAdd should be set
  And CapDrop should be set
  And DnsSearch should be set
  And Devices should be set
  And SecurityOpt should be set
  And WorkingDir should be set
  And Entrypoint should be set
  And User should be set
  And Hostname should be set
  And Domainname should be set
  And MacAddress should be set
  And Memory should be set
  And Privileged should be set
  And RestartPolicy should be set
  And RestartPolicy should be set
  And AttachStdin should be set
  And CpuShares should be set
  And CpusetCpus should be set
  And CpusetMems should be set
  And ReadonlyRootfs should be set
  And VolumeDriver should be set


# TODO: pid
# And VolumesFrom should be set
  # And MemorySwap should be set (done, no tests)
  # And Tty should be set (done, no tests)
