Given(/^there is another application linking to the named container$/) do
  response = create_application(
    goals: {
      service1: {
        image: "alpine:3.2",
        command: ["/bin/sh","-c","sleep 9999"],
        external_links: [
          "test_container:test"
        ]
      }
    },
    main_goal: 'service1'
  )
  expect(response.code).to eq(201)
end

Then(/^the second application should be running and be linked to the named container$/) do
  timed_retry do
    response = get_application
    expect(response.code).to eq(200)
    expect(response.to_h["goals"]["service1"]["status"]).to eq("running")
  end
  expect(inspect_goal("service1")["HostConfig"]["Links"]).to match([/test_container/])
end
