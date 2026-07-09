# frozen_string_literal: true
#
# Usage of RuboCop — the offense/cop framework, core cop set and
# .rubocop.yml configuration model added by `require "rubocop"`. It inspects
# a Ruby *source string* and reports offenses whose cop name, location and
# message byte-match the gem. Runs under go-embedded-ruby (rbgo); see
# examples/README.md.

require "rubocop"

source = <<~RUBY
  x = 1
  if x == 1 then
    puts "hi"
  end
RUBY

# The Runner (the commissioner) runs the default core cop set over the source.
runner = RuboCop::Runner.new
offenses = runner.inspect(source, "demo.rb")             # => Array of Offenses
p offenses.length                                        # => 3

# Each offense answers #cop_name, #message, #severity and #correctable?, and
# carries a 1-based #line / #column (also via its #location value object).
offenses.each do |o|
  puts "#{o.line}:#{o.column} #{o.severity} #{o.cop_name}"
  puts "  #{o.message}"
end

# The one-line report of an offense byte-matches RuboCop's simple format.
puts offenses.first.to_s      # => 1:1: C: [Correctable] Style/FrozenStringLiteralComment: ...

# #autocorrect returns a corrected copy of the source (here: the missing
# frozen_string_literal comment is prepended).
puts "---"
puts runner.autocorrect(source, "demo.rb")

# A .rubocop.yml disables a cop; a Runner built with that Config skips it.
config = RuboCop::Config.parse(<<~YML)
  Style/StringLiterals:
    Enabled: false
YML
names = RuboCop::Runner.new(config).inspect(source).map(&:cop_name)
p names                       # => ["Style/FrozenStringLiteralComment", "Style/IfUnlessModifier"]
