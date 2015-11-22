Given(/^I have a service depending on a service that will terminate$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 0.5; echo executed"]
      },
      service2: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 99999"],
        links: [
          'service1'
        ]
      }
    },
    main_goal: 'service2'
  )
  expect(response.code).to eq(201)
  timed_retry do
    response = get_application
    expect(response.to_h["goals"]["service2"]["status"]).to eq("running")
  end
end

When(/^the first service terminates$/) do
  timed_retry do
    response = get_application
    expect(response.to_h["goals"]["service1"]["status"]).to eq("terminated")
  end
end

Then(/^the second service should fail$/) do
  timed_retry do
    response = get_application
    expect(response.to_h["goals"]["service2"]["status"]).to eq("failed")
  end
end


Given(/^I have a service that fails after a timeout and is marked to restart$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 0.5; exit 1"],
        smart_restart: true
      }
    },
    main_goal: 'service1'
  )
  expect(response.code).to eq(201)
end

When(/^the service has failed$/) do
  timed_retry do
    expect(transition_log('service1').map{|x| x['status']}).to include('failed')
  end
end

Then(/^the service should be started again$/) do
  timed_retry do
    expect(transition_log('service1').map{|x| x['status']}.select{|x| ['failed', 'running'].include?(x)}).to eq(['running','failed', 'running'])
  end
end


Given(/^I have a service depending on a service that will terminate and both services are marked to restart$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 0.5; exit 1"],
        smart_restart: true
      },
      service2: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 99999"],
        links: [
          'service1'
        ],
        smart_restart: true
      }
    },
    main_goal: 'service2'
  )
  expect(response.code).to eq(201)
end

Then(/^the second service should be restarted$/) do
  timed_retry do
    expect(transition_log('service2').map{|x| x['status']}.select{|x| ['failed', 'running'].include?(x)}).to eq(['running','failed', 'running'])
  end
end

When(/^I have a task depending on a service that will terminate with service marked to restart$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 0.5; exit 1"],
        smart_restart: true
      },
      task1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 0.1"],
        links: [
          'service1'
        ]
      }
    },
    main_goal: 'task1'
  )
  expect(response.code).to eq(201)
end

Then(/^the task should be executed multiple times$/) do
  timed_retry retries: 30 do
    expect(transition_log('task1').map{|x| x['status']}.select{|x| ['terminated', 'running'].include?(x)}).to eq(['running','terminated', 'running', 'terminated'])
  end
end
