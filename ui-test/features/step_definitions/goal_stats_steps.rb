When(/^I navigate to application goal's stats$/) do
  visit('http://apparatchik:8080')
  click_link(@app_name)
  click_on('task1_transitions')
end

Then(/^I should see the goal's stats$/) do
  expect(page).to have_css('h4.modal-title')
end
