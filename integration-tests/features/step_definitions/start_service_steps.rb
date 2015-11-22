When(/^I create an application with one service$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 99999999"]
      }
    },
    main_goal: 'service1'
  )
  expect(response.code).to eq(201)
end

Then(/^the service should be running$/) do
  timed_retry do
    expect(get_application.to_h["goals"]["service1"]["status"]).to eq("running")
  end
  sleep 0.3
  expect(get_application.to_h["goals"]["service1"]["status"]).to eq("running")
end

When(/^I create an application with a service linked to another service$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 99999999"]
      },
      service2: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","grep service1 /etc/hosts; sleep 99999999"],
        links: ["service1"]
      },

    },
    main_goal: 'service2'
  )
  expect(response.code).to eq(201)
end

Then(/^both services should be running$/) do
  timed_retry do
    expect(get_application.to_h["goals"]["service1"]["status"]).to eq("running")
    expect(get_application.to_h["goals"]["service2"]["status"]).to eq("running")
  end
end

Then(/^second service should be linked to the first service$/) do
  response = inspect_goal('service2')
  expect(response.to_h['HostConfig']['Links']).to match([/service1$/])
end

When(/^I create an application with a service linked to another service with a link alias$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 99999999"]
      },
      service2: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","grep service1 /etc/hosts; sleep 99999999"],
        links: ["service1:awesome_service"]
      },

    },
    main_goal: 'service2'
  )
  expect(response.code).to eq(201)
end

Then(/^second service should be linked to the first service using the alias$/) do
  response = inspect_goal('service2')
  expect(response.to_h['HostConfig']['Links']).to match([/awesome_service$/])
end
