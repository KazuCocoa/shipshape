/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// This file uses the Clang style conventions.

#ifndef KYTHE_CXX_INDEXER_CXX_GRAPH_OBSERVER_H_
#define KYTHE_CXX_INDEXER_CXX_GRAPH_OBSERVER_H_

/// \file
/// \brief Defines the class kythe::GraphObserver

#include "clang/Basic/SourceLocation.h"
#include "clang/Basic/SourceManager.h"
#include "clang/Lex/Preprocessor.h"

#include "llvm/ADT/APSInt.h"
#include "llvm/ADT/StringRef.h"
#include "llvm/Support/raw_ostream.h"

namespace kythe {

// TODO(zarko): Most of the documentation for this interface belongs here.

/// \brief An interface for processing elements discovered as part of a
/// compilation unit.
///
/// Before calling a member on a `GraphObserver` that accepts a `SourceLocation`
/// as an argument, the `GraphObserver`'s `SourceManager` must be set.
class GraphObserver {
public:
  // TODO(zarko): Should GraphObserver be parameterized by
  // another type that provides the definitions of NodeId
  // and NameId (or just one of these)? Be explicit about
  // the guarantees about `signature`. There is some text
  // in the DESIGN file about it. Also, probably s/eId/eID/g.
  struct NodeId {
    std::string Signature;
    // \brief Returns a string representation of this `NodeId`.
    const std::string &ToString() const { return Signature; }
  };

  /// \brief A range of source text, potentially associated with a node.
  ///
  /// The `GraphObserver` interface uses `clang::SourceRange` instances to
  /// locate source text. It also supports associating a source range with
  /// a semantic context. This is used to distinguish generated code from the
  /// code that generates it. For instance, ranges specific to an implicit
  /// specialization of some type may have physical ranges that overlap with
  /// the primary template's, but they will be distinguished from the primary
  /// template's ranges by being associated with the context of the
  /// specialization.
  struct Range {
    enum class RangeKind {
      /// This range relates to a run of bytes in a source file.
      Physical,
      /// This range is related to some bytes, but also lives in an
      /// imaginary context; for example, a declaration of a member variable
      /// inside an implicit template instantiation has its source range
      /// set to the declaration's range in the template being instantiated
      /// and its context set to the `NodeId` of the implicit specialization.
      Wraith
    };
    /// \brief Constructs a physical `Range` for the given `clang::SourceRange`.
    explicit Range(const clang::SourceRange &R)
        : Kind(RangeKind::Physical), PhysicalRange(R) {}
    /// \brief Constructs a `Range` with some physical location, but specific to
    /// the context of some semantic node.
    Range(const clang::SourceRange &R, const NodeId &C)
        : Kind(RangeKind::Wraith), PhysicalRange(R), Context(C) {}
    /// \brief Constructs a new `Range` in the context of an existing
    /// `Range`, but with a different physical location.
    Range(const Range &R, const clang::SourceRange &NR)
        : Kind(R.Kind), PhysicalRange(NR), Context(R.Context) {}

    RangeKind Kind;
    clang::SourceRange PhysicalRange;
    NodeId Context;
  };

  struct NameId {
    /// \brief C++ distinguishes between several equivalence classes of names,
    /// a selection of which we consider important to represent.
    enum class NameEqClass {
      None,  ///< This name is not a member of a significant class.
      Union, ///< This name names a union record.
      Class  ///< This name names a struct or class.
      // TODO(zarko): Enums should be part of their own NameEqClass.
      // We should consider whether representation type information should
      // be part of enum (lookup) names--or whether this information must
      // be queried separately by clients and postprocessors.
    };
    /// \brief The abstract path to this name. For most `NameEqClass`es, this
    /// is the path from the translation unit root.
    std::string Path;
    /// \brief The `NameClass` of this name.
    NameEqClass EqClass;
    /// \brief Returns a string representation of this `NameId`.
    std::string ToString() const;
  };

  GraphObserver() {}

  /// \brief Sets the `SourceManager` that this `GraphObserver` should use.
  ///
  /// Since the `SourceManager` may not exist at the time the
  /// `GraphObserver` is created, it must be set after construction.
  ///
  /// \param SM the context for all `SourceLocation` instances.
  virtual void setSourceManager(clang::SourceManager *SM) {
    SourceManager = SM;
  }

  /// \param LO the language options in use.
  virtual void setLangOptions(clang::LangOptions *LO) { LangOptions = LO; }

  /// \param PP The `Preprocessor` to use.
  virtual void setPreprocessor(clang::Preprocessor *PP) { Preprocessor = PP; }

  /// \brief Returns the `NodeId` for the builtin type or type constructor named
  /// by `Spelling`.
  ///
  /// We invent names for type constructors, which are ordinarly spelled out
  /// as operators in C++. These names are "ptr" for *, "rref" for &&,
  /// "lref" for &, "const", "volatile", and "restrict".
  ///
  /// \param Spelling the spelling of the builtin type (from
  /// `clang::BuiltinType::getName(this->getLangOptions())` or the builtin
  /// type constructor.
  /// \return The `NodeId` for `Spelling`.
  virtual NodeId getNodeIdForBuiltinType(const llvm::StringRef &Spelling) = 0;

  /// \brief Returns the ID for a type node aliasing another type node.
  /// \param AliasName a `NameId` for the alias name.
  /// \param AliasedType a `NodeId` corresponding to the aliased type.
  /// \return the `NodeId` for the type node corresponding to the alias.
  virtual NodeId nodeIdForTypeAliasNode(const NameId &AliasName,
                                        const NodeId &AliasedType) = 0;

  /// \brief Records a type alias node (eg, from a `typedef` or
  /// `using Alias = ty` instance).
  /// \param AliasName a `NameId` for the alias name.
  /// \param AliasedType a `NodeId` corresponding to the aliased type.
  /// \return the `NodeId` for the type alias node this definition defines.
  virtual NodeId recordTypeAliasNode(const NameId &AliasName,
                                     const NodeId &AliasedType) = 0;

  /// \brief Returns the ID for a nominal type node (such as a struct,
  /// typedef or enum).
  /// \param TypeName a `NameId` corresponding to a nominal type.
  /// \return the `NodeId` for the type node corresponding to `TypeName`.
  virtual NodeId nodeIdForNominalTypeNode(const NameId &TypeName) = 0;

  /// \brief Records a type node for some nominal type (such as a struct,
  /// typedef or enum), returning its ID.
  /// \param TypeName a `NameId` corresponding to a nominal type.
  /// \return the `NodeId` for the type node corresponding to `TypeName`.
  virtual NodeId recordNominalTypeNode(const NameId &TypeName) = 0;

  /// \brief Records a type application node, returning its ID.
  /// \param TyconId The `NodeId` of the appropriate type constructor.
  /// \param Params The `NodeId`s of the types to apply to the constructor.
  /// \return The type application's result's ID.
  virtual NodeId recordTappNode(const NodeId &TyconId,
                                const std::vector<const NodeId *> &Params) = 0;

  enum class RecordKind { Struct, Class, Union };

  /// \brief Describes how autological a given declaration is.
  enum class Completeness {
    /// This declaration is a definition (`class C {};`) and hence is
    /// necessarily also complete.
    Definition,
    /// This declaration is complete (in the sense of [basic.types]),
    /// but it may not be a definition (`enum class E : short;`) [dcl.enum]
    Complete,
    /// This declaration is incomplete (`class C;`)
    Incomplete
  };

  /// \brief Records a node representing a record type (such as a class or
  /// struct).
  /// \param Node The NodeId of the record.
  /// \param Kind Whether this record is a struct, class, or union.
  /// \param RecordCompleteness Whether the record is complete.
  virtual void recordRecordNode(const NodeId &Node, RecordKind Kind,
                                Completeness RecordCompleteness) {}

  /// \brief Records a node representing a function.
  /// \param Node The NodeId of the function.
  /// \param FunctionCompleteness Whether the function is complete.
  virtual void recordFunctionNode(const NodeId &Node,
                                  Completeness FunctionCompleteness) {}

  /// \brief Records a node representing a callable, an object that can
  /// appear as the target of a call expression.
  /// \param Node The NodeId of the callable.
  ///
  /// Various language-level objects may be deemed 'callable' (like functions
  /// or classes with operator()). This abstraction allows call relationships
  /// to be recorded in the graph in a consistent way regardless of the
  /// particular kind of object being called.
  ///
  /// \sa recordCallableAsEdge
  virtual void recordCallableNode(const NodeId &Node) {}

  /// \brief Describes whether an enum is scoped (`enum class`).
  enum class EnumKind {
    Scoped,  ///< This enum is scoped (an `enum class`).
    Unscoped ///< This enum is unscoped (a plain `enum`).
  };

  /// \brief Records a node representing a dependent type abstraction, like
  /// a template.
  ///
  /// Abstraction nodes are used to represent the binding sites of compile-time
  /// variables. Consider the following class template definition:
  ///
  /// ~~~
  /// template <typename T> class C { T m; };
  /// ~~~
  ///
  /// Here, the `GraphObserver` will be notified of a single abstraction
  /// node. This node will have a single parameter, recorded with
  /// `recordAbsVarNode`. The abstraction node will have a child record
  /// node, which in turn will have a field `m` with a type that depends on
  /// the abstraction variable parameter.
  ///
  /// \param Node The NodeId of the abstraction.
  /// \sa recordAbsVarNode
  virtual void recordAbsNode(const NodeId &Node) {}

  /// \brief Records a node representing a variable in a dependent type
  /// abstraction.
  /// \param Node The `NodeId` of the variable.
  /// \sa recordAbsNode
  virtual void recordAbsVarNode(const NodeId &Node) {}

  /// \brief Records a node representing a deferred lookup.
  /// \param Node The `NodeId` of the lookup.
  /// \param Name The `Name` for which resolution has been deferred.
  virtual void recordLookupNode(const NodeId &Node,
                                const llvm::StringRef &Name) {}

  /// \brief Records a parameter relationship.
  /// \param `ParamOfNode` The node this `ParamNode` is the parameter of.
  /// \param `Ordinal` The ordinal for the parameter (0 is the first).
  /// \param `ParamNode` The `NodeId` for the parameter.
  virtual void recordParamEdge(const NodeId &ParamOfNode, uint32_t Ordinal,
                               const NodeId &ParamNode) {}

  /// \brief Records a node representing an enumerated type.
  /// \param Compl Whether the enum is complete.
  /// \param EnumKind Whether the enum is scoped.
  virtual void recordEnumNode(const NodeId &Node, Completeness Compl,
                              EnumKind Kind) {}

  /// \brief Records a node representing a constant with a value representable
  /// with an integer.
  ///
  /// Note that the type of the language-level object this node represents may
  /// not be integral. For example, `recordIntegerConstantNode` is used to
  /// record information about the implicitly or explicitly assigned values
  /// for enumerators.
  virtual void recordIntegerConstantNode(const NodeId &Node,
                                         const llvm::APSInt &Value) {}
  // TODO(zarko): Handle other values. How should we represent class
  // constants?

  /// \brief Records that a variable (either local or global) has been
  /// declared.
  /// \param DeclName The name to which this element is being bound.
  /// \param DeclNode The identifier for this particular element.
  /// \param Compl The completeness of this variable declaration.
  // TODO(zarko): We should make note of the storage-class-specifier (dcl.stc)
  // of the variable, which is a property the variable itself and not of its
  // type.
  virtual void recordVariableNode(const NameId &DeclName,
                                  const NodeId &DeclNode, Completeness Compl) {}

  // TODO(zarko): recordExpandedTypeEdge -- records that a type was seen
  // to have some canonical type during a compilation. (This is a 'canonical'
  // type for a given compilation, but may differ between compilations.)

  /// \brief Records that a particular `Range` contains the definition
  /// of the node called `DefnId`.
  ///
  /// Generally the `Range` provided will be small and limited only to
  /// the part of the declaration that binds a name. For example, in `class C`,
  /// we would `recordDefinitionRange` on the range for `C`.
  virtual void recordDefinitionRange(const Range &SourceRange,
                                     const NodeId &DefnId) {}

  /// \brief Describes how specific a completion relationship is.
  enum class Specificity {
    /// This relationship is the only possible relationship given its context.
    /// For example, a class definition uniquely completes a forward declaration
    /// in the same source file.
    UniquelyCompletes,
    /// This relationship is one of many possible relationships. For example, a
    /// forward declaration in a header file may be completed by definitions in
    /// many different source files.
    Completes
  };

  /// \brief Records that a particular `Range` contains a completion
  /// for the node named `DefnId`.
  /// \param SourceRange The source range containing the completion.
  /// \param DefnId The `NodeId` for the node being completed.
  /// \param Spec the specificity of the relationship beween the `Range`
  /// and the `DefnId`.
  virtual void recordCompletionRange(const Range &SourceRange,
                                     const NodeId &DefnId, Specificity Spec) {}

  /// \brief Records that a particular `Node` has been given some `Name`.
  ///
  /// A given node may have zero or more names distinct from its `NodeId`.
  /// These may be used as entry points into the graph. For example,
  /// `recordNamedEdge` may be called with the unique `NodeId` for a function
  /// definition and the `NameId` corresponding to the fully-qualified name
  /// of that function; it may subsequently be called with that same `NodeId`
  /// and a `NameId` corresponding to that function's mangled name.
  ///
  /// A call to `recordNamedEdge` may have the following form:
  /// ~~~
  /// // DeclName is the lookup name for some record type (roughly, the "C" in
  /// // "class C").
  /// GraphObserver::NameId DeclName = BuildNameIdForDecl(RecordDecl);
  /// // DeclNode refers to a particular decl of some record type.
  /// GraphObserver::NodeId DeclNode = BuildNodeIdForDecl(RecordDecl);
  /// Observer->recordNamedEdge(DeclNode, DeclName);
  /// ~~~
  virtual void recordNamedEdge(const NodeId &Node, const NameId &Name) {}

  /// \brief Records the type of a node as an edge in the graph.
  /// \param TermNodeId The identifier for the node to be given a type.
  /// \param TypeNodeId The identifier for the node representing the type.
  virtual void recordTypeEdge(const NodeId &TermNodeId,
                              const NodeId &TypeNodeId) {}

  /// \brief Records that some term specializes some type.
  /// \param TermNodeId The identifier for the node specializing the type.
  /// \param TypeNodeId The identifier for the node representing the specialized
  /// type.
  virtual void recordSpecEdge(const NodeId &TermNodeId,
                              const NodeId &TypeNodeId) {}

  /// \brief Records that one node participates in the call graph as a
  /// particular `Callable`.
  ///
  /// This relationship allows the indexer to abstract the notion of
  /// callability from its embodiment in particular callable objects
  /// (such as functions or records that define operator()).
  ///
  /// \param ToCallId The specific node that may be called.
  /// \param CallableAsId The node representing `ToCallId` in the call graph.
  virtual void recordCallableAsEdge(const NodeId &ToCallId,
                                    const NodeId &CallableAsId) {}

  /// \brief Records that a callable is called at a particular location.
  /// \param CallLoc The `Range` responsible for making the call.
  /// \param CallerId The scope to be held responsible for making the call;
  /// for example, a function.
  /// \param CalleeId The callable being called.
  virtual void recordCallEdge(const Range &SourceRange, const NodeId &CallerId,
                              const NodeId &CalleeId) {}

  /// \brief Records a child-to-parent relationship as an edge in the graph.
  /// \param ChildNodeId The identifier for the child node.
  /// \param ParentNodeId The identifier for the parent node.
  virtual void recordChildOfEdge(const NodeId &ChildNodeId,
                                 const NodeId &ParentNodeId) {}

  /// \brief Records a use site for some decl.
  virtual void recordDeclUseLocation(const Range &SourceRange,
                                     const NodeId &DeclId) {}

  /// \brief Records that a type was spelled out at a particular location.
  /// \param SourceRange The source range covering the type spelling.
  /// \param TypeNode The identifier for the type being spelled out.
  virtual void recordTypeSpellingLocation(const Range &SourceRange,
                                          const NodeId &TypeId) {}

  /// \brief Called when a new input file is entered.
  ///
  /// The file entered in the first `pushFile` is the compilation unit being
  /// indexed.
  ///
  /// \param Loc A `SourceLocation` in the file being entered.
  /// \sa popFile
  virtual void pushFile(clang::SourceLocation Location) {}

  /// \brief Called when the previous input file to be entered is left.
  /// \sa pushFile
  virtual void popFile() {}

  virtual ~GraphObserver() = 0;

  clang::SourceManager *getSourceManager() { return SourceManager; }

  clang::LangOptions *getLangOptions() { return LangOptions; }

  clang::Preprocessor *getPreprocessor() { return Preprocessor; }

protected:
  clang::SourceManager *SourceManager = nullptr;
  clang::LangOptions *LangOptions = nullptr;
  clang::Preprocessor *Preprocessor = nullptr;
};

inline GraphObserver::~GraphObserver() {}

/// \brief A GraphObserver that does nothing.
class NullGraphObserver : public GraphObserver {
public:
  NodeId getNodeIdForBuiltinType(const llvm::StringRef &Spelling) override {
    return NodeId();
  }

  NodeId nodeIdForTypeAliasNode(const NameId &AliasName,
                                const NodeId &AliasedType) override {
    return NodeId();
  }

  NodeId recordTypeAliasNode(const NameId &AliasName,
                             const NodeId &AliasedType) override {
    return NodeId();
  }

  NodeId nodeIdForNominalTypeNode(const NameId &type_name) override {
    return NodeId();
  }

  NodeId recordNominalTypeNode(const NameId &TypeName) override {
    return NodeId();
  }

  NodeId recordTappNode(const NodeId &TyconId,
                        const std::vector<const NodeId *> &Params) override {
    return NodeId();
  }

  ~NullGraphObserver() {}
};

/// \brief Emits a stringified representation of the given `NameId`,
/// including its `NameEqClass`, to the given stream.
inline llvm::raw_ostream &operator<<(llvm::raw_ostream &OS,
                                     const GraphObserver::NameId &N) {
  OS << N.Path << "#";
  switch (N.EqClass) {
  case GraphObserver::NameId::NameEqClass::None:
    OS << "n";
    break;
  case GraphObserver::NameId::NameEqClass::Class:
    OS << "c";
    break;
  case GraphObserver::NameId::NameEqClass::Union:
    OS << "u";
    break;
  }
  return OS;
}

/// \brief Emits a stringified representation of the given `NodeId`.
inline llvm::raw_ostream &operator<<(llvm::raw_ostream &OS,
                                     const GraphObserver::NodeId &N) {
  return OS << N.Signature;
}

inline std::string GraphObserver::NameId::ToString() const {
  std::string Rep;
  llvm::raw_string_ostream Ostream(Rep);
  Ostream << *this;
  return Ostream.str();
}

inline bool operator==(const GraphObserver::Range &L,
                       const GraphObserver::Range &R) {
  return L.Kind == R.Kind && L.PhysicalRange == R.PhysicalRange &&
         (L.Kind == GraphObserver::Range::RangeKind::Physical ||
          L.Context.Signature == R.Context.Signature);
}

inline bool operator!=(const GraphObserver::Range &L,
                       const GraphObserver::Range &R) {
  return !(L == R);
}

} // namespace kythe

#endif // KYTHE_CXX_INDEXER_CXX_GRAPH_OBSERVER_H_
