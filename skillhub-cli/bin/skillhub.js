#!/usr/bin/env node

const { Command } = require('commander');
const chalk = require('chalk');
const fs = require('fs-extra');
const path = require('path');
const { execSync } = require('child_process');

const SKILLS_DIR = path.join(process.env.HOME, '.hermes', 'skills');
const REGISTRY_DIR = path.join(process.env.HOME, '.hermes', 'skillhub-registry');

// 内置技能源（本地目录）
const LOCAL_REGISTRY = path.join(process.env.HOME, 'aispace', 'hermes-agent', 'skills');

const program = new Command();

program
  .name('skillhub')
  .description('Hermes Agent Skill Hub - 安装、搜索、管理 Agent 技能包')
  .version('1.0.0');

// 搜索技能
program
  .command('search <keyword>')
  .description('搜索可用技能包')
  .option('-r, --registry <path>', '指定注册表路径')
  .action(async (keyword, options) => {
    const registry = options.registry || LOCAL_REGISTRY;
    console.log(chalk.cyan(`\n🔍 搜索技能包: "${keyword}"\n`));
    
    const results = [];
    
    // 搜索本地已安装的技能
    if (await fs.pathExists(SKILLS_DIR)) {
      const installed = await fs.readdir(SKILLS_DIR);
      for (const name of installed) {
        if (name.startsWith('.')) continue;
        const skillPath = path.join(SKILLS_DIR, name, 'SKILL.md');
        if (await fs.pathExists(skillPath)) {
          const content = await fs.readFile(skillPath, 'utf8');
          if (content.toLowerCase().includes(keyword.toLowerCase()) || 
              name.toLowerCase().includes(keyword.toLowerCase())) {
            results.push({ name, status: 'installed', content });
          }
        }
      }
    }
    
    // 搜索本地注册表（可用但未安装的）
    if (await fs.pathExists(registry)) {
      const available = await fs.readdir(registry);
      for (const name of available) {
        if (name.startsWith('.')) continue;
        const isInstalled = results.some(r => r.name === name);
        if (!isInstalled) {
          const skillPath = path.join(registry, name, 'SKILL.md');
          if (await fs.pathExists(skillPath)) {
            const content = await fs.readFile(skillPath, 'utf8');
            if (content.toLowerCase().includes(keyword.toLowerCase()) ||
                name.toLowerCase().includes(keyword.toLowerCase())) {
              results.push({ name, status: 'available', content });
            }
          }
        }
      }
    }
    
    if (results.length === 0) {
      console.log(chalk.yellow('未找到匹配的技能包'));
      return;
    }
    
    console.log(chalk.gray(`找到 ${results.length} 个结果:\n`));
    results.forEach((r, i) => {
      const icon = r.status === 'installed' ? chalk.green('✅') : chalk.blue('📦');
      const status = r.status === 'installed' ? chalk.green('已安装') : chalk.blue('可安装');
      console.log(`${icon} ${chalk.bold(r.name)} (${status})`);
      
      // 提取描述
      const descMatch = r.content.match(/description:\s*(.+)/i);
      if (descMatch) {
        console.log(`   ${chalk.gray(descMatch[1].trim().substring(0, 80))}`);
      }
      console.log();
    });
  });

// 安装技能
program
  .command('install <name>')
  .alias('i')
  .description('安装技能包')
  .option('-s, --source <path>', '技能源路径')
  .option('-f, --force', '强制覆盖已安装的版本')
  .action(async (name, options) => {
    const source = options.source || LOCAL_REGISTRY;
    const destPath = path.join(SKILLS_DIR, name);
    const sourcePath = path.join(source, name);
    
    console.log(chalk.cyan(`\n📦 安装技能包: ${name}\n`));
    
    // 检查是否已安装
    if (await fs.pathExists(destPath) && !options.force) {
      console.log(chalk.yellow(`⚠️  技能包 "${name}" 已安装`));
      console.log(chalk.gray(`   使用 --force 覆盖安装`));
      return;
    }
    
    // 检查源是否存在
    if (!await fs.pathExists(sourcePath)) {
      console.log(chalk.red(`❌ 找不到技能包 "${name}"`));
      console.log(chalk.gray(`   搜索可用技能: skillhub search ${name}`));
      return;
    }
    
    try {
      // 备份旧版本
      if (await fs.pathExists(destPath)) {
        const backupPath = destPath + '.bak.' + Date.now();
        await fs.move(destPath, backupPath);
        console.log(chalk.gray(`   备份旧版本到: ${backupPath}`));
      }
      
      // 复制技能包
      await fs.copy(sourcePath, destPath);
      console.log(chalk.green(`✅ 安装成功: ${name}`));
      console.log(chalk.gray(`   位置: ${destPath}`));
      
      // 显示技能信息
      const skillPath = path.join(destPath, 'SKILL.md');
      if (await fs.pathExists(skillPath)) {
        const content = await fs.readFile(skillPath, 'utf8');
        const descMatch = content.match(/description:\s*(.+)/i);
        if (descMatch) {
          console.log(chalk.gray(`   描述: ${descMatch[1].trim()}`));
        }
      }
    } catch (err) {
      console.log(chalk.red(`❌ 安装失败: ${err.message}`));
    }
  });

// 卸载技能
program
  .command('uninstall <name>')
  .alias('rm')
  .description('卸载技能包')
  .option('-f, --force', '跳过确认')
  .action(async (name, options) => {
    const destPath = path.join(SKILLS_DIR, name);
    
    console.log(chalk.cyan(`\n🗑️  卸载技能包: ${name}\n`));
    
    if (!await fs.pathExists(destPath)) {
      console.log(chalk.yellow(`⚠️  技能包 "${name}" 未安装`));
      return;
    }
    
    if (!options.force) {
      const readline = require('readline').createInterface({
        input: process.stdin,
        output: process.stdout
      });
      
      const answer = await new Promise(resolve => {
        readline.question(chalk.yellow('确定卸载此技能包？(y/N) '), resolve);
      });
      readline.close();
      
      if (answer.toLowerCase() !== 'y') {
        console.log(chalk.gray('取消卸载'));
        return;
      }
    }
    
    try {
      await fs.remove(destPath);
      console.log(chalk.green(`✅ 已卸载: ${name}`));
    } catch (err) {
      console.log(chalk.red(`❌ 卸载失败: ${err.message}`));
    }
  });

// 列出已安装技能
program
  .command('list')
  .alias('ls')
  .description('列出已安装的技能包')
  .action(async () => {
    console.log(chalk.cyan('\n📋 已安装的技能包:\n'));
    
    if (!await fs.pathExists(SKILLS_DIR)) {
      console.log(chalk.yellow('技能目录不存在'));
      return;
    }
    
    const skills = await fs.readdir(SKILLS_DIR);
    const installed = [];
    
    for (const name of skills) {
      if (name.startsWith('.')) continue;
      const skillPath = path.join(SKILLS_DIR, name, 'SKILL.md');
      if (await fs.pathExists(skillPath)) {
        const content = await fs.readFile(skillPath, 'utf8');
        const descMatch = content.match(/description:\s*(.+)/i);
        const categoryMatch = content.match(/category:\s*(.+)/i);
        installed.push({
          name,
          description: descMatch ? descMatch[1].trim() : '',
          category: categoryMatch ? categoryMatch[1].trim() : 'other'
        });
      }
    }
    
    if (installed.length === 0) {
      console.log(chalk.yellow('暂无已安装的技能包'));
      return;
    }
    
    // 按分类分组
    const grouped = {};
    installed.forEach(s => {
      if (!grouped[s.category]) grouped[s.category] = [];
      grouped[s.category].push(s);
    });
    
    for (const [category, skills] of Object.entries(grouped)) {
      console.log(chalk.bold(`\n📁 ${category}`));
      skills.forEach(s => {
        console.log(`  ${chalk.green('✅')} ${chalk.bold(s.name)}`);
        if (s.description) {
          console.log(`     ${chalk.gray(s.description.substring(0, 60))}`);
        }
      });
    }
    
    console.log(chalk.gray(`\n共 ${installed.length} 个技能包`));
  });

// 更新技能
program
  .command('update <name>')
  .description('更新技能包到最新版本')
  .option('-s, --source <path>', '技能源路径')
  .action(async (name, options) => {
    console.log(chalk.cyan(`\n🔄 更新技能包: ${name}\n`));
    await program.parseAsync(['node', 'skillhub', 'install', name, '--force', ...(options.source ? ['-s', options.source] : [])]);
  });

// 更新所有技能
program
  .command('update-all')
  .description('更新所有已安装的技能包')
  .option('-s, --source <path>', '技能源路径')
  .action(async (options) => {
    console.log(chalk.cyan('\n🔄 更新所有技能包\n'));
    
    if (!await fs.pathExists(SKILLS_DIR)) {
      console.log(chalk.yellow('技能目录不存在'));
      return;
    }
    
    const skills = await fs.readdir(SKILLS_DIR);
    for (const name of skills) {
      if (name.startsWith('.')) continue;
      const skillPath = path.join(SKILLS_DIR, name, 'SKILL.md');
      if (await fs.pathExists(skillPath)) {
        await program.parseAsync(['node', 'skillhub', 'update', name, ...(options.source ? ['-s', options.source] : [])]);
      }
    }
  });

program.parse(process.argv);
