When(/^the service is running$/) do
  timed_retry do
    expect(get_application.to_h["goals"]["service1"]["status"]).to eq("running")
  end
end

Then(/^I should be able to inspect the goal container of the service$/) do
  response = inspect_goal('service1')
  expect(response.code).to eq(200)
  expect(response.to_h).to have_key('Id')
  expect(response.to_h).to have_key('Created')
end

When(/^I inspect a goal of an non\-existing task$/) do
  @response = inspect_goal('servicex')
end

Then(/^the service should respond with (\d+) status code$/) do |expected|
  expect(@response.code).to eq(expected.to_i)
end

When(/^I inspect a goal of a non\-existing application$/) do
  @app_name = 'whatever'
  @response = inspect_goal('servicex')
end
