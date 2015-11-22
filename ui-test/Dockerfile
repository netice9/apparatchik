FROM ruby:2.2.2
RUN apt-get update -qq && apt-get install -y build-essential libpq-dev cmake npm
RUN ln -s /usr/bin/nodejs /usr/bin/node
RUN npm install -g phantomjs
RUN gem install bundler -v 1.10.5
RUN mkdir /tests
WORKDIR /tests
ADD Gemfile /tests/Gemfile
ADD Gemfile.lock /tests/Gemfile.lock
RUN bundle install -j 8
ADD . /tests
CMD []
