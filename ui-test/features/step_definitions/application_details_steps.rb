When(/^I click on details of that application$/) do
  visit('http://apparatchik:8080')
  click_link(@app_name)
end

Then(/^I should see the name of the application$/) do
  expect(find('#application_name').text).to eq(@app_name)
end

Then(/^I should see the name of the application's main goal$/) do
  expect(find('#main_goal').text).to eq('task1')

end


Then(/^I should see application's goals$/) do
  expect(all('.goal-name').map(&:text)).to eq(['task1'])
end
