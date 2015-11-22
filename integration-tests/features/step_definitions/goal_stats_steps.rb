Given(/^I create an application with one service that uses some cpu$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","while true; do echo test ; sleep 0.1; done;"]
      }
    },
    main_goal: 'service1'
  )
  expect(response.code).to eq(201)end

When(/^I wait for the service to start$/) do
  timed_retry do
    response = get_application
    expect(response.code).to eq(200)
    expect(response.to_h["goals"]["service1"]["status"]).to eq("running")
  end
end

Then(/^the service should have some cpu and memory stats$/) do
  expect(goal_stats('service1').code).to eq(200)

  timed_retry retries: 30 do
    expect(goal_stats('service1').to_h['cpu_stats']).not_to be_empty
  end

  timed_retry do
    expect(goal_stats('service1').to_h['mem_stats']).not_to be_empty
  end

end

Then(/^the service should have current stats$/) do
  timed_retry do
    expect(goal_current_stats('service1').code).to eq(200)
  end

  expect(goal_current_stats('service1').to_h).to have_key('read')

end
