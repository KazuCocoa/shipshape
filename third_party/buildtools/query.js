'use strict';

var DEFAULT_BUILD_KIND = 'build';

/**
 * Returns the set of targets that are dependencies of the given target/targets.
 */
var deps = function() {
  function dfs(targets, action) {
    var discovered = {};
    while (targets.length > 0) {
      var current = targets.pop();
      if (!discovered[current.id]) {
        discovered[current.id] = true;
        action(current);
        for (var i = 0; i < current.inputs.length; ++i) {
          if (current.inputs[i].json) {
            targets.push(current.inputs[i]);
          }
        }
      }
    }
  }
  return function(targets) {
    targets = resolveTargets(targets);
    var deparr = [];
    dfs(targets, function(node) {
      deparr.push(node);
    });
    return deparr;
  };
}();

function filterByKind(kind) {
  return function(target) {
    // see if a parent (outputs) has this target
    // as an input with the appropriate kind.
    if (!target.outputs || target.outputs.length === 0) {
      return false;
    }
    for (var i = 0; i < target.outputs.length; ++i) {
      var candidates = target.outputs[i].inputsByKind[kind] || [];
      for (var j = 0; j < candidates.length; ++j) {
        if (candidates[j].id == target.id) {
          return true;
        }
      }
    }
    return false;
  };
}

/**
 * Returns a subset of targets that has rule kinds matching {@code pattern}.
 */
function kind(pattern, targets) {
  targets = resolveTargets(targets);
  return targets.filter(function(target) {
    return target.rule &&
        target.rule.config_name &&
        target.rule.config_name.match(pattern);
  });
}

/**
 * Returns the set of files outputs for the given target(s).
 */
function outputs(targets, kind) {
  kind = kind || DEFAULT_BUILD_KIND;
  targets = resolveTargets(targets);
  var outs = [];
  for (var i = 0; i < targets.length; i++) {
    var target = targets[i];
    outs = outs.concat(target.rule.getOutputsFor(target, kind));
  }
  return outs;
}

/**
 * Returns the set of file inputs for the given target(s).
 */
function files(targets) {
  targets = resolveTargets(targets);
  var res = [];
  for (var i = 0; i < targets.length; i++) {
    var inputs = targets[i].inputs;
    for (var j = 0; j < inputs.length; j++) {
      if (!inputs[j].json) {
        res.push(inputs[j]);
      }
    }
  }
  return res;
}

/**
 * Returns the set of targets that depends on the given target(s)/file(s).
 */
function dependsOn(targets) {
  targets = resolveTargets(targets);
  var resSet = {};
  var work = targets;
  while (work.length > 0) {
    var added = [];
    for (var i = 0; i < work.length; i++) {
      var target = work[i];
      for (var j = 0; j < target.outputs.length; j++) {
        var output = target.outputs[j];
        if (!resSet[output.id]) {
          resSet[output.id] = output;
          added.push(output);
        }
      }
    }
    work = added;
  }

  var res = [];
  for (var id in resSet) {
    res.push(resSet[id]);
  }
  return res;
}

// Internal function used to resolve a target string specification or array of
// specifications to an array of resolved target configurations.
function resolveTargets(targets) {
  targets = Array.isArray(targets) ? targets : [targets];
  return targets.map(function(target) {
    if (typeof target != 'string') {
      return [target];
    }
    var rt = global.query_engine.resolveTargets(target);
    if (Array.isArray(rt)) {
      return rt;
    }
    return [rt];
  }).reduce(function(p, n) { return p.concat(n); }, []);
}

// Query engine entry point.  Executes the given query, returning the resolved
// set of target results.
exports.queryEval = function(query) {
  // Fully resolve targets graph
  var all = global.query_engine.resolveTargets('//...');
  for (var i = 0; i < all.length; i++) {
    if (all[i].rule.getBuilds) {
      all[i].rule.getBuilds(all[i]);
    }
  }

  try {
    return resolveTargets(eval(query));
  } catch (err) {
    console.error(err);
    return undefined;
  }
};
