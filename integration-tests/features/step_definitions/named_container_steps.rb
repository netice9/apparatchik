Given(/^there is one application with a named container$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 999999"],
        container_name: "test_container"
      }
    },
    main_goal: 'service1'
  )
  expect(response.code).to eq(201)
end

Then(/^a container with the service's name should exist$/) do
  container = Docker::Container.all(:all => true).map(&:json).find{|c| c['Name'] == '/test_container'}
  expect(container).not_to be_nil
end


When(/^I start an application with a service linked to another service in the same application that has a named container$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 999999"],
        container_name: "test_container"
      },
      service2: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 999999"],
        links: ['service1']
      }
    },
    main_goal: 'service2'
  )
  expect(response.code).to eq(201)
end

Then(/^the second service should be running$/) do
  timed_retry do
    response = get_application
    expect(response.code).to eq(200)
    expect(response.to_h["goals"]["service2"]["status"]).to eq("running")
  end
end
