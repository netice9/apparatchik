require 'httparty'
require 'pry'
require 'securerandom'
require 'docker'

Before do
  applications = HTTParty.get("http://apparatchik:8080/api/v1.0/applications").to_a
  applications.each do |application_name|
    HTTParty.delete("http://apparatchik:8080/api/v1.0/applications/#{application_name}")
  end
end

After ("~@wip")do
  applications = HTTParty.get("http://apparatchik:8080/api/v1.0/applications").to_a
  applications.each do |application_name|
    HTTParty.delete("http://apparatchik:8080/api/v1.0/applications/#{application_name}")
  end
end
