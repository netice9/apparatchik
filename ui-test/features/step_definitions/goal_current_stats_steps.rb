When(/^I navigate to application goal's current stats$/) do
  visit('http://apparatchik:8080')
  click_link(@app_name)
  click_on('task1_stats')
end

Then(/^I should see the goal's current stats$/) do
  timed_retry do
    visit('http://apparatchik:8080')
    click_link(@app_name)
    click_on('task1_stats')
    expect(page).to have_css('h4.modal-title')
  end
end
