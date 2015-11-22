When(/^I click on Create Application$/) do
  visit('http://apparatchik:8080')
  click_link('New Application')
end

When(/^I fill in a new name for the application$/) do
  fill_in 'Application Name', with: 'uploaded_app'
end

When(/^I fill in a valid application description file$/) do
  attach_file('file', File.expand_path("../../../fixtures/application.json", __FILE__) )
end

When(/^I upload the application$/) do
  click_on 'Create'
end

Then(/^the new application should be running$/) do
  # expect(all('#active_applications a').map(&:text)).to eq(['uploaded_app'])
  expect(page).to have_selector('#active_applications a', text: 'uploaded_app')
end


