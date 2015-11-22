When(/^I navigate to output of the application's goal$/) do
  visit('http://apparatchik:8080')
  click_link(@app_name)
  click_on('task1_logs')
end

Then(/^I should see the output$/) do
  expect(page).to have_css('h4.modal-title')
end
