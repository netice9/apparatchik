Given(/^I create an application with one task that will execute succesfully$/) do
  response = create_application(
    goals: {
      task1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 1; echo executed"]
      }
    },
    main_goal: 'task1'
  )
  expect(response.to_h).to match(
    {
      "name"=>"#{@app_name}",
      "goals"=> {
        "task1"=>{
          "name"=>"task1",
          "status"=>an_instance_of(String)
        }
      },
      "main_goal"=>"task1"
    }
  )
  expect(response.code).to eq(201)
end

When(/^I wait for the task to finish$/) do
  timed_retry do
    response = get_application
    expect(response.code).to eq(200)
    expect(response.to_h["goals"]["task1"]["status"]).to eq("terminated")
  end
end

Then(/^the exit code of the task should be (\d+)$/) do |expected_code|
  expect(get_application.to_h["goals"]["task1"]["exit_code"]).to eq(expected_code.to_i)
end

Then(/^I should be able to retreive the logs of the task$/) do
  response = HTTParty.get("http://apparatchik:8080/applications/#{@app_name}/task1/logs")
  expect(response.code).to eq(200)
  expect(response.headers['content-type']).to eq('text/plain')
  expect(response.body).to eq("executed\n")
end


Given(/^I create an application with one task depending on another task$/) do
response = create_application(
  goals: {
      task1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","echo executed 1"]
      },
      task2: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","echo executed 2"],
        run_after: ["task1"]
      }
    },
    main_goal: 'task2'
  )
  expect(response.code).to eq(201)
end

Given(/^I wait for the second task to finish$/) do
  timed_retry retries: 20 do
    response = get_application
    expect(response.to_h["goals"]["task2"]["status"]).to eq("terminated")
  end
end

Then(/^the exit code of the first task should be (\d+)$/) do |expected_code|
  expect(get_application.to_h["goals"]["task1"]["exit_code"]).to eq(expected_code.to_i)
end

Then(/^the exit code of the second task should be (\d+)$/) do |expected_code|
  expect(get_application.to_h["goals"]["task2"]["exit_code"]).to eq(expected_code.to_i)
end

Given(/^I create an application with one task depending on another task with first task failing$/) do
  response = create_application(
    goals: {
      task1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","exit 1"]
      },
      task2: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","echo executed 2"],
        run_after: ["task1"]
      }
    },
    main_goal: 'task2'
  )

  expect(response.code).to eq(201)
end

Given(/^I wait for the first task to finish$/) do
  timed_retry do
    expect(get_application.to_h["goals"]["task1"]["status"]).to eq("terminated")
  end
end

Given(/^I wait for the first task to fail$/) do
  timed_retry do
    expect(get_application.to_h["goals"]["task1"]["status"]).to eq("failed")
  end
end

Then(/^the second task should be waiting for successful execution of the first task$/) do
  sleep 0.2
  expect(get_application.to_h["goals"]["task2"]["status"]).to eq("waiting_for_dependencies")
end

When(/^I delete the application$/) do
  response = HTTParty.delete("http://apparatchik:8080/applications/#{@app_name}")
  expect(response.code).to eq(204)
end


Then(/^the application should not exist anymore$/) do
  expect(get_application.code).to eq(404)
end
