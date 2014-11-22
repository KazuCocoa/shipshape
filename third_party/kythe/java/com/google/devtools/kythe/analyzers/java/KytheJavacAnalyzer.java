package com.google.devtools.kythe.analyzers.java;

import com.google.common.base.Preconditions;
import com.google.devtools.kythe.analyzers.base.FactEmitter;
import com.google.devtools.kythe.platform.java.JavaCompilationDetails;
import com.google.devtools.kythe.platform.java.JavacAnalyzer;
import com.google.devtools.kythe.platform.java.helpers.SignatureGenerator;
import com.google.devtools.kythe.platform.shared.AnalysisException;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;

import com.sun.source.tree.CompilationUnitTree;
import com.sun.tools.javac.api.JavacTaskImpl;
import com.sun.tools.javac.tree.JCTree.JCCompilationUnit;
import com.sun.tools.javac.util.Context;

import java.io.IOException;
import java.nio.charset.Charset;

/** {@link JavacAnalyzer} to emit Kythe nodes and edges. */
public class KytheJavacAnalyzer extends JavacAnalyzer {

  private final FactEmitter emitter;

  // should be set in analyzeCompilationUnit before any call to analyzeFile
  private JavaEntrySets entrySets;

  public KytheJavacAnalyzer(FactEmitter emitter) {
    Preconditions.checkArgument(emitter != null, "FactEmitter must be non-null");
    this.emitter = emitter;
  }

  @Override
  public void analyzeCompilationUnit(JavaCompilationDetails details) throws AnalysisException {
    Preconditions.checkState(entrySets == null,
        "JavaEntrySets is non-null (analyzeCompilationUnit was called concurrently?)");
    CompilationUnit compilation = details.getCompilationUnit();
    entrySets =
        new JavaEntrySets(emitter, compilation.getVName(), compilation.getRequiredInputList());
    try {
      super.analyzeCompilationUnit(details);
    } finally {
      entrySets = null; // Ensure entrySets is cleared for error-checking
    }
  }

  @Override
  public void analyzeFile(JavaCompilationDetails details, CompilationUnitTree ast)
      throws AnalysisException {
    Preconditions.checkState(entrySets != null,
        "analyzeCompilationUnit must be called to analyze each file");
    Context context = ((JavacTaskImpl) details.getJavac()).getContext();
    SignatureGenerator signatureGenerator = new SignatureGenerator(ast, context);
    try {
      KytheTreeScanner.emitEntries(context, entrySets, signatureGenerator,
          (JCCompilationUnit) ast, Charset.forName(details.getEncoding()));
    } catch (IOException e) {
      throw new AnalysisException("Exception analyzing file: " + ast.getSourceFile().getName(), e);
    }
  }
}
