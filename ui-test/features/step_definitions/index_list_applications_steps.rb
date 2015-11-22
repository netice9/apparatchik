When(/^there are no applications running by apparatchik$/) do
end

When(/^I visit the index page$/) do
  visit('http://apparatchik:8080')
end

Then(/^I should see empty list of applications$/) do
  expect(page).to have_css('#active_applications')
  expect(all('#active_applications a').count).to eq(0)
end

When(/^there is one application run by apparatchik$/) do
  response = create_application({
      goals: {
        task1: {
          image: "alpine:3.2",
          command: ["/bin/sh","-c","sleep 1; echo executed"],
          task: true
        }
      },
      main_goal: 'task1'
    }
  )
end

Then(/^I should see the running application$/) do
  expect(page).to have_css('#active_applications')
  expect(all('#active_applications a').count).to eq(1)
end
