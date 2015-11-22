def create_application( request )
  @app_name = SecureRandom.uuid
  @response = HTTParty.put("http://apparatchik:8080/api/v1.0/applications/#{@app_name}",
    body: request.to_json,
    headers: {
        'Content-Type' => 'application/json',
        'Accept' => 'application/json'
    },
    no_follow: true,
    timeout: 5
  )

  timed_retry sleep: 0.5, retries: 240 do
    get_application.to_h["goals"].each_pair do |name, goal|
      expect(goal["status"]).not_to eq("fetching_image")
    end
  end

  @response
end

def get_application
  HTTParty.get("http://apparatchik:8080/api/v1.0/applications/#{@app_name}",
    headers: {
        'Accept' => 'application/json'
    }
  )
end

def inspect_goal(goal_name)
  HTTParty.get("http://apparatchik:8080/api/v1.0/applications/#{@app_name}/goals/#{goal_name}/inspect",
    headers: {
        'Accept' => 'application/json'
    }
  )
end

def transition_log(goal_name)
  HTTParty.get("http://apparatchik:8080/api/v1.0/applications/#{@app_name}/goals/#{goal_name}/transition_log",
    headers: {
        'Accept' => 'application/json'
    }
  )
end


def goal_stats(goal_name)
  HTTParty.get("http://apparatchik:8080/api/v1.0/applications/#{@app_name}/goals/#{goal_name}/stats",
    headers: {
        'Accept' => 'application/json'
    }
  )
end

def goal_current_stats(goal_name)
  HTTParty.get("http://apparatchik:8080/api/v1.0/applications/#{@app_name}/goals/#{goal_name}/current_stats",
    headers: {
        'Accept' => 'application/json'
    }
  )
end