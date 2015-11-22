FROM ruby:2.2.0
RUN gem install bundler -v 1.10.5
RUN mkdir /tests
WORKDIR /tests
ADD Gemfile /tests/Gemfile
ADD Gemfile.lock /tests/Gemfile.lock
RUN bundle install -j 4
ADD . /tests
ENTRYPOINT ["/bin/bash"]
