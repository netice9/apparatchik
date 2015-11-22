Then(/^I should be able to inspect the goal transition log$/) do
  response = transition_log('service1')
  expect(response.code).to eq(200)
  expect(response.to_a.map{|x| x['status'] }).to eq(["fetching_image", "starting", "running"])
  expect(response.to_a.map{|x| Time.iso8601(x['time']) }).to match([an_instance_of(Time)] * 3)
end

When(/^I inspect transition log an non\-existing goal$/) do
  @response = transition_log('serviceX')
end

When(/^I inspect transition log of a non\-existing application$/) do
  @app_name = 'not_existing'
  @response = transition_log('serviceX')
end
