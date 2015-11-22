When(/^I delete the application$/) do
  visit('http://apparatchik:8080')
  click_link(@app_name)
  click_on 'Delete'
  click_on 'delete_confirm'
end
