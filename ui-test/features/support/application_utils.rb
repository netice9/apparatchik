def create_application( request, app_name = nil )

  @app_name = app_name || SecureRandom.uuid

  @response = HTTParty.put("http://apparatchik:8080/applications/#{@app_name}",
    body: request.to_json,
    headers: {
        'Content-Type' => 'application/json',
        'Accept' => 'application/json'
    }
  )

  timed_retry sleep: 0.5, retries: 240 do
    get_application.to_h["goals"].each_pair do |name, goal|
      expect(goal["status"]).not_to eq("fetching_image")
    end
  end

  @response
end

def get_application
  HTTParty.get("http://apparatchik:8080/applications/#{@app_name}",
    headers: {
        'Accept' => 'application/json'
    }
  )
end

def inspect_goal(goal_name)
  HTTParty.get("http://apparatchik:8080/applications/#{@app_name}/#{goal_name}/inspect",
    headers: {
        'Accept' => 'application/json'
    }
  )
end

def transition_log(goal_name)
  HTTParty.get("http://apparatchik:8080/applications/#{@app_name}/#{goal_name}/transition_log",
    headers: {
        'Accept' => 'application/json'
    }
  )
end