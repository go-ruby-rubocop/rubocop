# rubocop examples

Runnable pure-Ruby usage of RuboCop's offense/cop framework, core cop set and `.rubocop.yml` configuration model, verified under the [rbgo](https://github.com/go-embedded-ruby) interpreter.

```sh
rbgo examples/rubocop_usage.rb
```

| File | Shows |
| --- | --- |
| `rubocop_usage.rb` | Run the default core cop set over a source string with `RuboCop::Runner#inspect`, read each `RuboCop::Cop::Offense` (`#cop_name` / `#message` / `#severity` / `#line` / `#column` / `#location` / `#correctable?` / `#to_s`), rewrite the source with `RuboCop::Runner#autocorrect`, and disable a cop via a `.rubocop.yml` parsed with `RuboCop::Config.parse`. |
