Given(/^there are no applications$/) do
end

When(/^I list the applications$/) do
  response = HTTParty.get("http://apparatchik:8080/applications")
  expect(response.code).to eq(200)
  @applications = response.to_a
end

Then(/^I should get an empty list$/) do
  expect(@applications).to eq([])
end

Given(/^there is one application$/) do
  response = create_application(
    goals: {
      task1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 1; echo executed"]
      }
    },
    main_goal: 'task1'
  )
  expect(response.code).to eq(201)
end

Then(/^I should get a list with only that application$/) do
  expect(@applications).to eq([@app_name])
end
