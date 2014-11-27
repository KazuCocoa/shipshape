#!/bin/bash
# This script runs the indexer on various test cases, piping the results
# to the verifier. The test cases contain assertions for the verifier to
# verify. Should every case succeed, this script returns zero.
HAD_ERRORS=0
VERIFIER=campfire-out/bin/third_party/kythe/cxx/verifier/verifier
INDEXER=campfire-out/bin/third_party/kythe/cxx/indexer/cxx/indexer
BASEDIR=third_party/kythe/cxx/indexer/cxx/testdata
# one_case test-file clang-standard verifier-argument indexer-argument
function one_case {
  ${INDEXER} -i $1 $4 -- -std=$2 | ${VERIFIER} $1 $3
  RESULTS=( ${PIPESTATUS[0]} ${PIPESTATUS[1]} )
  if [ ${RESULTS[0]} -ne 0 ]; then
    echo "[ FAILED INDEX: $1 ]"
    HAD_ERRORS=1
  elif [ ${RESULTS[1]} -ne 0 ]; then
    echo "[ FAILED VERIFY: $1 ]"
    HAD_ERRORS=1
  else
    echo "[ OK: $1 ]"
  fi
}

# Remember to add these files to CAMPFIRE as well.
one_case "${BASEDIR}/empty_case.cc" "c++1y"
one_case "${BASEDIR}/alias_alias_int.cc" "c++1y"
one_case "${BASEDIR}/alias_alias_ptr_int.cc" "c++1y"
one_case "${BASEDIR}/alias_and_cvr.cc" "c++1y"
one_case "${BASEDIR}/alias_int.cc" "c++1y"
one_case "${BASEDIR}/alias_int_twice.cc" "c++1y"
one_case "${BASEDIR}/anchor_utf8.cc" "c++1y"
one_case "${BASEDIR}/enum_class_decl.cc" "c++1y"
one_case "${BASEDIR}/enum_class_element_decl.cc" "c++1y"
one_case "${BASEDIR}/enum_decl.cc" "c++1y"
one_case "${BASEDIR}/enum_decl_completes.cc" "c++1y"
one_case "${BASEDIR}/enum_decl_ty.cc" "c++1y"
one_case "${BASEDIR}/enum_decl_ty_completes.cc" "c++1y"
one_case "${BASEDIR}/enum_decl_ty_header_completes.cc" "c++1y"
one_case "${BASEDIR}/enum_element_decl.cc" "c++1y"
one_case "${BASEDIR}/file_content.cc" "c++1y"
one_case "${BASEDIR}/file_node.cc" "c++1y"
one_case "${BASEDIR}/file_node_reentrant.cc" "c++1y"
one_case "${BASEDIR}/function_args_decl.cc" "c++1y"
one_case "${BASEDIR}/function_args_defn.cc" "c++1y"
one_case "${BASEDIR}/function_decl.cc" "c++1y"
one_case "${BASEDIR}/function_decl_completes.cc" "c++1y"
one_case "${BASEDIR}/function_defn.cc" "c++1y"
one_case "${BASEDIR}/function_defn_call.cc" "c++1y"
one_case "${BASEDIR}/function_direct_call.cc" "c++1y"
one_case "${BASEDIR}/function_knr_ty.c" "c99"
one_case "${BASEDIR}/function_operator_overload_names.cc" "c++1y" --ignore_dups=true
one_case "${BASEDIR}/function_operator_parens.cc" "c++1y"
one_case "${BASEDIR}/function_operator_parens_call.cc" "c++1y" --ignore_dups=true
one_case "${BASEDIR}/function_operator_parens_overload.cc" "c++1y"
one_case "${BASEDIR}/function_operator_parens_overload_call.cc" "c++1y" --ignore_dups=true
one_case "${BASEDIR}/function_overload.cc" "c++1y"
one_case "${BASEDIR}/function_overload_call.cc" "c++1y"
one_case "${BASEDIR}/function_ptr_ty.cc" "c++1y"
one_case "${BASEDIR}/function_ty.cc" "c++1y"
one_case "${BASEDIR}/function_vararg.cc" "c++1y"
one_case "${BASEDIR}/function_vararg_ty.cc" "c++1y"
one_case "${BASEDIR}/rec_anon_struct.cc" "c++1y"
one_case "${BASEDIR}/rec_class.cc" "c++1y"
one_case "${BASEDIR}/rec_class_header_completes.cc" "c++1y"
one_case "${BASEDIR}/rec_class_macro.cc" "c++1y"
one_case "${BASEDIR}/rec_struct.c" "c99"
one_case "${BASEDIR}/rec_struct.cc" "c++1y"
one_case "${BASEDIR}/rec_struct_decl.cc" "c++1y"
one_case "${BASEDIR}/rec_union.cc" "c++1y"
one_case "${BASEDIR}/template_alias_implicit_instantiation.cc" "c++1y"
one_case "${BASEDIR}/template_arg_multiple_typename.cc" "c++1y"
one_case "${BASEDIR}/template_arg_typename.cc" "c++1y"
one_case "${BASEDIR}/template_class_defn.cc" "c++1y"
one_case "${BASEDIR}/template_class_inst_implicit.cc" "c++1y"
one_case "${BASEDIR}/template_class_inst_explicit.cc" "c++1y"
one_case "${BASEDIR}/template_class_inst_implicit_dependent.cc" "c++1y"
one_case "${BASEDIR}/template_class_skip_implicit_on.cc" "c++1y" "" "--index_template_instantiations=false"
one_case "${BASEDIR}/template_depname_class.cc" "c++1y"
one_case "${BASEDIR}/template_depname_inst_class.cc" "c++1y"
one_case "${BASEDIR}/template_depname_path_graph.cc" "c++1y"
one_case "${BASEDIR}/template_fn_decl.cc" "c++1y"
one_case "${BASEDIR}/template_fn_decl_defn.cc" "c++1y"
one_case "${BASEDIR}/template_fn_defn.cc" "c++1y"
one_case "${BASEDIR}/template_fn_explicit_spec_completes.cc" "c++1y"
one_case "${BASEDIR}/template_fn_explicit_spec_with_default_completes.cc" "c++1y"
one_case "${BASEDIR}/template_fn_implicit_spec.cc" "c++1y"
one_case "${BASEDIR}/template_fn_multi_decl_def.cc" "c++1y"
one_case "${BASEDIR}/template_fn_multiple_implicit_spec.cc" "c++1y"
one_case "${BASEDIR}/template_fn_overload.cc" "c++1y"
one_case "${BASEDIR}/template_fn_spec.cc" "c++1y"
one_case "${BASEDIR}/template_fn_spec_decl.cc" "c++1y"
one_case "${BASEDIR}/template_fn_spec_defn_decl.cc" "c++1y"
one_case "${BASEDIR}/template_fn_spec_defn_defn_decl_decl.cc" "c++1y"
one_case "${BASEDIR}/template_fn_spec_overload.cc" "c++1y"
one_case "${BASEDIR}/template_instance_type_from_class.cc" "c++1y"
one_case "${BASEDIR}/template_multilevel_argument.cc" "c++1y"
one_case "${BASEDIR}/template_ps_completes.cc" "c++1y"
one_case "${BASEDIR}/template_ps_decl.cc" "c++1y"
one_case "${BASEDIR}/template_ps_defn.cc" "c++1y"
one_case "${BASEDIR}/template_ps_multiple_decl.cc" "c++1y"
one_case "${BASEDIR}/template_ps_twovar_decl.cc" "c++1y"
one_case "${BASEDIR}/template_ty_typename.cc" "c++1y"
one_case "${BASEDIR}/template_two_arg_spec.cc" "c++1y"
one_case "${BASEDIR}/typedef_class_anon_ns.cc" "c++1y"
one_case "${BASEDIR}/typedef_class.cc" "c++1y"
one_case "${BASEDIR}/typedef_class_nested_ns.cc" "c++1y"
one_case "${BASEDIR}/typedef_const_int.cc" "c++1y"
one_case "${BASEDIR}/typedef_int.cc" "c++1y"
one_case "${BASEDIR}/typedef_nested_class.cc" "c++1y"
one_case "${BASEDIR}/typedef_paren.cc" "c++1y"
one_case "${BASEDIR}/typedef_ptr_int_canonicalized.cc" "c++1y"
one_case "${BASEDIR}/typedef_ptr_int.cc" "c++1y"
one_case "${BASEDIR}/typedef_same.cc" "c++1y"
one_case "${BASEDIR}/vardecl_double_shadowed_local_anchor.cc" "c++1y"
one_case "${BASEDIR}/vardecl_global_anchor.cc" "c++1y"
one_case "${BASEDIR}/vardecl_global_anon_ns_anchor.cc" "c++1y"
one_case "${BASEDIR}/vardecl_global_tu_anchor.cc" "c++1y"
one_case "${BASEDIR}/vardecl_local_anchor.cc" "c++1y"
one_case "${BASEDIR}/vardecl_shadowed_local_anchor.cc" "c++1y"
one_case "${BASEDIR}/wild_std_allocator.cc" "c++1y"

exit ${HAD_ERRORS}
