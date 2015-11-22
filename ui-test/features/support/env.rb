require 'capybara/cucumber'
require 'capybara/poltergeist'
require 'httparty'
require 'pry'
Capybara.javascript_driver = :poltergeist
Capybara.default_driver = :poltergeist

Before do
  unless @connected
    timed_retry do
      expect(HTTParty.get('http://apparatchik:8080').code).to eq(200)
    end
    @connected = true
  end

  applications = HTTParty.get("http://apparatchik:8080/applications").to_a
  applications.each do |application_name|
    HTTParty.delete("http://apparatchik:8080/applications/#{application_name}")
  end

end