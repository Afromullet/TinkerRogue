# The Besieged Stronghold: Garrison + Attrition Combined Design

**Status:** Design Draft
**Parent:** `docs/dungeon_cavern_brainstorm.md` (Ideas 1 & 4)

---

## 1. Core Premise

The dungeon is an active enemy stronghold with a **finite, organized garrison** that the player must grind through across multiple floors -- but the player's own squads **degrade over time**. Enemies don't respawn, but neither does the player's strength fully recover between fights. Both sides are burning down. The central question every turn is: *who is losing faster, and can I shift that balance?*

This creates a **double resource curve**. The garrison's defensive strength is a declining staircase -- each cleared patrol or post is permanently gone. The player's combat readiness is a sawtooth wave -- it drops during fights, partially recovers at safe rooms, then drops again. Victory goes to the player who manages the interaction between these two curves: spending resources efficiently against the garrison while preserving enough to reach the bottom.

---

## 2. The Garrison

### 2.1 Enemy Organization

The garrison is not a random scattering of enemies. It is a structured military presence organized into functional groups, each with a role in the stronghold's defense.

**Patrols** move along fixed routes through corridors and connecting areas. They are the garrison's eyes. A patrol that spots the player (or finds evidence of combat -- bloodstains, missing guards, open doors that should be closed) triggers an alert escalation. Patrols are typically light units: fast, moderate damage, low durability. Their purpose is detection, not engagement -- though they will fight if cornered.

**Guard Posts** are stationary defenders at chokepoints, doorways, and intersections. They are the garrison's skeleton -- the fixed structure everything else hangs on. Guard posts are typically medium units with defensive positioning (behind barricades, in elevated positions, flanking doorways). Eliminating a guard post opens a route permanently, but the fight is often in the defenders' favor due to positioning.

**Reserves** are off-duty enemies in barracks, mess halls, or training rooms. They are slower to mobilize but represent the garrison's depth. At low alert, reserves are vulnerable -- off-guard, unarmored, possibly asleep. At high alert, they gear up and reinforce threatened areas. The reserve pool is the garrison's insurance policy: even if the player clears the outer defenses efficiently, reserves can fill gaps.

**Command Structure** consists of officers, sergeants, and other leadership entities scattered through the garrison. They provide passive buffs to nearby units (morale, coordination, damage) and are the nodes through which alert information propagates. Killing officers degrades local garrison effectiveness -- nearby units lose buffs, patrols may become disorganized, and alert escalation slows in that sector. Officers are high-priority targets but are usually positioned behind other defenders.

**Specialists** are unique enemies tied to specific rooms or functions. The armory has a quartermaster guarding equipment. The alchemy lab has a poisoner who rigs traps. The kennel has a beastmaster with war animals. Specialists give rooms tactical identity and provide distinctive encounters that break up the patrol-post-reserve rhythm.

### 2.2 Alert Level

The garrison operates under a **dungeon-wide alert level** that rises as the player is detected, fights break out, or evidence is discovered. Alert is a one-way ratchet within a floor -- it only goes up. (Cross-floor alert is discussed in Section 6.)

| Alert Tier | Trigger Threshold | Garrison Behavior |
|---|---|---|
| **Unaware** | Default | Normal routines. Patrols follow standard routes. Reserves are off-duty. Guards are at posts but not at full readiness. |
| **Suspicious** | Minor evidence (open door, missing guard not yet confirmed) | Patrol routes tighten. Guards become more attentive (wider detection radius). Some reserves begin gearing up. |
| **Alerted** | Confirmed contact (patrol reports back, alarm sounded) | Patrols shift to active search patterns. Reserves mobilize and move to reinforce key positions. Officers may reposition to coordinate defense. Guard posts receive reinforcements. |
| **Lockdown** | Heavy losses or high-value target threatened | Remaining garrison collapses to defensible positions around critical areas (treasury, commander, floor exit). Patrols cease -- all units hunker down. Barricades go up. Traps are armed in abandoned corridors. |

**Alert propagation is not instant.** It travels through the garrison's communication network. If the player eliminates a patrol before it can report, no alert is raised. If a runner is sent to warn the next floor but the player intercepts them, that floor stays unaware. Alert propagation has a physical reality -- it moves through corridors, requires living messengers (or alarm bells, signal horns, magical links depending on faction), and can be disrupted.

**Alert decay does not happen within a floor.** Once the garrison knows something is wrong, they don't forget. However, if the player clears a floor quickly and moves down before runners escape, lower floors may remain at lower alert levels.

### 2.3 Room Types and Tactical Meaning

Rooms are not just containers for enemies. Each room type communicates its function to the player and implies tactical considerations.

- **Barracks / Sleeping Quarters:** High enemy count, low readiness at low alert. At high alert, the barracks is empty because reserves have deployed. Clearing it early removes the reserve pool; clearing it late finds it already abandoned.
- **Mess Hall / Common Area:** Off-duty enemies in a social configuration. Moderate count, slow reaction time. Good ambush opportunity. May contain consumable supplies (food that restores fatigue).
- **Armory:** Guarded by well-equipped specialists. Contains gear upgrades and replacement equipment. The garrison draws from the armory when alert rises -- clearing it early means reinforcements are worse equipped.
- **Officer's Quarters / Command Room:** Contains command structure. Heavily defended but high-value: killing officers degrades coordination across the floor. May contain intelligence (map fragments, patrol schedules) that reveals hidden rooms or patrol routes.
- **Alchemy Lab / Workshop:** Contains consumable resources (potions, repair kits, trap components). Guarded by a specialist who may use the room's contents defensively (throwing potions, activating workshop traps).
- **Treasury / Vault:** Deep in the stronghold, behind multiple layers of defense. Contains the floor's best loot. At Lockdown alert, the garrison's last stand happens here.
- **Kennel / Beast Pens:** War animals that are released at higher alert levels. Clearing the kennel before alert rises prevents beasts from joining later fights.
- **Shrine / Ritual Chamber:** Faction-specific. May provide buffs to the garrison or debuffs to the player. Desecrating the shrine removes the effect but may have alert or morale consequences.
- **Safe Rooms** (see Section 3.3): Hidden alcoves, abandoned cells, collapsed sections -- places the garrison doesn't patrol. These are where the player rests.

---

## 3. The Attrition Layer

### 3.1 What Degrades

The player's squads are not infinitely renewable combat machines. Multiple resource axes degrade over the course of a dungeon run, and they recover at different rates and through different means.

**Hit Points** are the most visible resource. Damage taken in combat reduces HP. Partial healing is available through consumables and safe room rest, but full HP recovery within a dungeon is rare. Squads accumulate wounds over multiple fights.

**Fatigue** is a per-unit resource that increases with combat and movement. Fatigued units suffer stat penalties (reduced accuracy, slower initiative, lower damage). Fatigue recovers with rest but is the primary cost of "just one more fight" -- even a squad at full HP may be too fatigued to fight effectively.

**Consumables** are finite supplies brought into the dungeon and found during exploration. Healing potions, antidotes, repair kits, buff items. They cannot be crafted mid-run (unless the player finds and holds an alchemy lab). Every consumable used is gone. This creates save-or-spend tension: use the potion now on a moderate wound, or save it for a crisis that may never come?

**Spell Charges / Ability Uses** represent limited-use powers that don't fully recharge between fights. A mage might have 5 fireballs for the entire floor. A healer's restoration spell has 3 uses before needing extended rest. This prevents the player from relying on a single powerful ability as a crutch and forces varied tactical approaches across encounters.

**Morale** is a squad-level resource that reflects cumulative stress. Taking casualties, fighting while fatigued, encountering elite enemies, and being surprised all reduce morale. Low morale causes penalties to initiative, accuracy, and may trigger panic or rout in extreme cases. Morale recovers slowly with rest and rises when the squad achieves victories or finds safe rooms.

**Equipment Durability** degrades with use. Weapons lose effectiveness, armor provides less protection. Repair kits restore durability but are consumable. Finding replacement equipment in the garrison's armory becomes tactically valuable, not just a loot opportunity.

### 3.2 Degradation Rates

Not all resources degrade at the same pace. This creates a shifting bottleneck that changes which resource the player is most worried about.

- **Early floors:** Consumables and spell charges are plentiful. HP and fatigue are the main concerns. The player fights freely but accumulates small wounds.
- **Mid floors:** Consumables are noticeably depleted. Spell charges are running low. The player starts rationing and choosing fights more carefully. Fatigue management becomes important -- squad rotation matters.
- **Deep floors:** Everything is scarce. The player has partially wounded squads, limited consumables, few special abilities remaining, and equipment showing wear. Every fight must be weighed against the cost. This is where garrison attrition pays off -- if the player cleared efficiently above, fewer enemies remain below.

### 3.3 Safe Rooms

Safe rooms are locations the garrison doesn't patrol -- hidden alcoves, collapsed sections, abandoned cells, secret passages. They are the player's oases in hostile territory.

**What resting provides:**
- Partial HP recovery (percentage-based, not full heal)
- Significant fatigue reduction
- Moderate morale recovery
- NO consumable restoration (you use what you brought or found)
- NO spell charge restoration (partial at best -- maybe 1 charge per rest)
- Equipment does not self-repair

**What resting costs:**
Resting is not free. While the player's squads rest, the garrison continues to operate. The specific costs depend on the current alert level:

| Alert Level | Rest Consequence |
|---|---|
| **Unaware** | Minimal. Patrols continue normal routes. Small chance a patrol discovers evidence of the player's passage during the rest period. |
| **Suspicious** | Patrols tighten routes, increasing the chance of stumbling on the safe room's vicinity. Some reserves begin mobilizing. |
| **Alerted** | Active search parties sweep areas the player has been. Reinforcements move to key positions. The safe room itself is secure, but the corridors around it may now have enemies that weren't there before. |
| **Lockdown** | The garrison consolidates. Resting doesn't make things worse (enemies are already in defensive positions) but it doesn't help either -- the player emerges from rest to face a fortified enemy. |

The rest-vs-push tradeoff is the attrition system's central decision. Pushing forward while strong means fighting at higher effectiveness but accumulating fatigue and wounds that compound later. Resting means recovering but giving the garrison time to react.

### 3.4 Squad Rotation

The player can bring **multiple squads** into a dungeon. Only one squad (or a limited number) is active at a time; the others rest in safe rooms or hang back at cleared areas. This creates a rotation strategy:

- **Primary squad** fights the current set of encounters until degraded
- **Secondary squad** swaps in while the primary rests
- **Specialist squad** is reserved for specific encounter types (e.g., a squad built for anti-mage combat saved for the ritual chamber)

Rotation is the player's primary tool for managing attrition. A player with three well-built squads can sustain combat readiness much longer than one with a single squad, even if that single squad is individually stronger.

**Rotation constraints:**
- Swapping squads requires reaching a safe room or cleared area (no mid-combat swaps)
- Squads that are resting can be ambushed if the safe room is discovered (unlikely but possible at high alert)
- Moving inactive squads through the dungeon still costs exploration turns and risks detection

---

## 4. The Synergy: Compound Decisions

The garrison and attrition systems are individually interesting, but their interaction is where the design creates genuinely difficult, rewarding decisions. Neither system alone produces these dilemmas.

### 4.1 The Rest Paradox

Resting recovers the player's resources but gives the garrison time to react. This is not a simple "rest is bad" dynamic -- it depends on alert level and garrison composition.

**At low alert**, resting is nearly free. The garrison doesn't know the player is there. But the player probably doesn't need to rest yet -- they're still fresh. The temptation is to push forward while strong, which causes wounds and fatigue, which means they'll need to rest later when resting is more expensive.

**At high alert**, resting is costly because the garrison is actively repositioning. But this is exactly when the player is most wounded and exhausted from the fights that caused that alert. The player needs rest most when rest costs most. This is the core tension.

**Implication:** The player is rewarded for fighting efficiently (fewer fights, faster kills, less alert generated per garrison strength removed) because efficient fighting means less need for costly resting.

### 4.2 The Engagement Economy

Every fight has three cost dimensions:

1. **Squad resources spent** (HP, fatigue, consumables, spell charges)
2. **Alert generated** (detection, noise, evidence left behind)
3. **Garrison strength removed** (enemies killed, positions cleared)

The ideal fight maximizes (3) while minimizing (1) and (2). But these goals often conflict:

- Using powerful abilities kills enemies fast (high 3, low 1 for HP) but may consume limited spell charges (high 1 for charges)
- A drawn-out cautious fight conserves special abilities (low 1 for charges) but accumulates fatigue and HP damage (high 1 for fatigue/HP) and gives enemies more time to trigger alarms (high 2)
- Avoiding a fight entirely costs nothing (0 for 1 and 2) but leaves enemies alive who may appear as reinforcements later (0 for 3, future cost)
- An ambush that wipes the enemy before they react is ideal but requires approach route planning and sometimes burning consumables for positioning (buff items, smoke bombs)

### 4.3 The Priority Target Problem

Certain garrison elements are high-value targets whose removal disproportionately weakens the garrison:

- **Officers** degrade coordination across a sector
- **The armory** prevents equipment upgrades for reinforcing reserves
- **The kennel** removes war beasts from all future encounters
- **Alarm mechanisms** (bells, horns, signal fires) slow alert propagation

But these targets are defended. Reaching them costs resources. The player must decide: spend resources now on a high-value target to make everything else easier, or conserve resources and grind through standard encounters?

This decision is further complicated by attrition. A fresh squad can assault the armory with acceptable losses. A fatigued squad assaulting the same target takes heavier casualties, uses more consumables, and generates more alert (longer fight, more chances for alarms). The cost of a priority target assault rises as attrition accumulates, which means **the best time to hit priority targets is early, when the player is strongest** -- but early is also when the player is least certain about the floor's layout.

### 4.4 The Information Game

The player operates under fog of war. They don't know the garrison's layout until they explore. But exploration risks detection. Scouting generates information but also alert.

- **Fast scouts** (light units with high movement) can reveal the map quickly but can't fight patrols they encounter
- **Cautious probing** (the full squad moves slowly, ready for combat) generates less alert but reveals less per turn
- **Found intelligence** (patrol schedules from officer's quarters, maps from the command room) can reveal parts of the floor without physical exploration

Information has compounding value in this combined system. Knowing where the armory is lets the player plan an efficient route. Knowing patrol timings lets the player rest between patrols without detection. Knowing where safe rooms are lets the player plan rotation points. The player who invests in information gathering early can make better decisions throughout the floor.

### 4.5 The Attrition Tipping Point

There exists a critical threshold where the player's accumulated attrition exceeds their ability to handle remaining encounters efficiently. Beyond this point, each fight costs disproportionately more resources because:

- Fatigued units miss more, extending fights
- Wounded units take damage they can't afford, consuming scarce healing
- Low morale causes panic, disrupting tactics
- Depleted spell charges remove tactical options, forcing brute-force approaches

This creates a **death spiral** if the player overcommits. The design must provide escape valves:

- **Retreat is always possible.** The player can abandon the floor and return to the overworld. They keep loot found so far. The garrison partially recovers (some reserves return to posts, alert drops one tier on the cleared floor) but dead enemies stay dead. This means partial runs still have value.
- **Resource caches exist.** Hidden rooms with consumables, alchemy labs with potion ingredients, armories with replacement gear. Finding these can pull a struggling run back from the brink.
- **Squad rotation absorbs spikes.** A fresh secondary squad can handle an encounter that would destroy the fatigued primary squad.

---

## 5. Reward Design

### 5.1 Reward Categories

Loot in a garrison-attrition dungeon falls into categories with different tactical values depending on the player's current state.

**Recovery Items** (healing potions, fatigue restoratives, morale boosters, repair kits) are the most consistently valuable loot in this system. In a standard dungeon, a healing potion is a minor convenience. In an attrition dungeon, it is a tactical asset that extends the squad's operational range by one more fight. Recovery items are distributed throughout the dungeon but concentrated off the critical path -- the player must spend exploration effort (and risk detection) to find them.

**Equipment Upgrades** are found in garrison armories, officer's quarters, and specialist rooms. They provide permanent power increases for the run. A weapon upgrade means every subsequent fight is slightly easier, which means less resource drain per fight, which compounds over the remaining floors. Early equipment finds are disproportionately valuable.

**Intelligence** (maps, patrol schedules, keys, command documents) provides information that translates into efficiency. A map revealing the floor layout saves scouting turns and reduces detection risk. A key to the back entrance of the treasury lets the player bypass its front-door defenders. Intelligence is found in command rooms, officer's quarters, and by interrogating (or looting) officer enemies.

**Currency and Valuables** (gold, gems, trade goods) have no immediate in-dungeon value but represent overworld progression. They are the "greed" reward -- pursuing them costs dungeon resources for overworld benefit. They are concentrated in the treasury and scattered in minor caches throughout.

**Unique / Rare Items** are the dungeon's signature rewards. The commander's personal weapon, a faction-specific artifact, a rare crafting material. These are behind the hardest fights or deepest exploration. They are the reason to push past the attrition tipping point -- if you can survive to reach them.

### 5.2 Reward Placement Philosophy

Rewards are placed to create decisions, not just accumulate:

- **Critical path** has enough recovery items to sustain a straight shot to the floor exit, but nothing more. The player who rushes gets through but gains little.
- **Side paths** have equipment upgrades, extra recovery items, and intelligence. They cost exploration effort and risk detection but make the rest of the floor easier.
- **Deep optional areas** have currency, valuables, and rare items. They are for the player who has managed their resources well enough to go off-script.
- **Garrison-specific rooms** (armory, alchemy lab, kennel) contain thematically appropriate loot and serve a dual purpose: the loot is valuable, and clearing the room weakens the garrison.

### 5.3 The Recovery Item Economy

Recovery items are the critical currency of the attrition system. Their distribution follows a deliberate curve:

- **Brought from overworld:** The player's starting supply. Determined by overworld preparation. Better-prepared expeditions start with more.
- **Found on early floors:** Moderate supply. Enough to establish a comfortable pace if the player explores.
- **Found on mid floors:** Diminishing supply. The dungeon gives less as the player needs more.
- **Found on deep floors:** Scarce. Recovery items on deep floors are in dangerous optional areas. The player who needs healing most must fight hardest to find it.

This curve means the player's recovery item supply is front-loaded. Efficient play on early floors (taking less damage, finding more caches) creates a surplus that sustains deeper exploration. Sloppy early play depletes the supply early and creates a deficit that compounds.

---

## 6. Floor Progression

### 6.1 Multi-Floor Structure

A garrison dungeon has 3-5 floors (scalable based on desired length). Each floor is a self-contained tactical map with its own garrison composition, room layout, and safe room placement. The player completes one floor, descends, and faces the next.

### 6.2 Cross-Floor Persistence

**Player-side persistence:**
- Squad HP, fatigue, morale, equipment durability carry over
- Consumable counts carry over
- Spell charges may partially recover between floors (long rest at the floor transition, representing the descent)
- The player can choose which squads to bring to the next floor (leaving damaged squads behind to guard the retreat)

**Garrison-side persistence:**
- Dead enemies stay dead across the entire run
- Alert level propagates downward: if a runner escapes a floor, the next floor starts at Suspicious or Alerted
- If the player clears a floor before runners escape, the next floor starts Unaware
- Lower floors have stronger garrison compositions (better-equipped, higher-tier enemies, more officers) but the same organizational structure

### 6.3 Escalation Across Floors

Each floor escalates both the garrison's strength and the attrition pressure:

| Floor | Garrison Character | Attrition Pressure | Typical Player State |
|---|---|---|---|
| **Floor 1** | Outer defenses. Light patrols, basic guard posts, small reserve. Alarm mechanisms that warn deeper floors. | Low. Player is fresh. Main concern is fighting efficiently to preserve long-term resources. | Full strength. Learning the dungeon's patterns. |
| **Floor 2** | Inner perimeter. Heavier patrols, fortified posts, larger reserve pool. Officers present. Specialist rooms appear. | Moderate. Accumulated wounds and fatigue from Floor 1. Consumable supply noticeably reduced. | Partially degraded. Squad rotation becomes important. |
| **Floor 3** | Core garrison. Elite units, multiple officers, war beasts deployed (if kennel wasn't cleared). Defensive architecture favors the garrison. | High. Resources are strained. Every fight matters. Recovery items are precious. | Significantly degraded. Efficient play on prior floors pays dividends here. |
| **Floor 4+** | Commander's domain. The garrison's best, in their strongest positions. Boss encounter at the floor's heart. Minimal patrol activity -- everything is a fortified position. | Critical. The player is pushing through on reserves. Only well-managed runs reach this point in fighting shape. | Running on fumes or still viable, depending entirely on prior floor performance. |

### 6.4 The Runner Mechanic

When alert hits Alerted on a floor, the garrison attempts to send **runners** to the next floor down. Runners are fast, lightly armed enemies who flee toward the floor exit. Their only goal is to escape.

- If the runner escapes: the next floor starts at Suspicious or Alerted. Reserves are pre-mobilized, patrols are on heightened routes, ambush positions are prepared.
- If the player intercepts the runner: the next floor starts Unaware. The full garrison advantage is available.

This creates a high-stakes mini-objective within each floor. The player must balance fighting the current threats with intercepting runners. A runner slipping past can be more costly than losing a squad member, because the consequences cascade through all remaining floors.

---

## 7. Overworld Integration

### 7.1 Faction Flavor

The garrison's faction determines the dungeon's aesthetic, enemy composition, and mechanical twists on the core systems.

**Bandit Stronghold:**
- Garrison is disorganized but cunning. Fewer patrols, more traps and ambush positions.
- Alert mechanics include alarm bells and signal fires. Physical infrastructure the player can disable.
- Safe rooms are hidden smuggler's caches. Recovery items include stolen goods and black-market supplies.
- Attrition pressure comes from traps and poison (ongoing damage between fights) rather than raw combat power.
- Overworld effect: Clearing a bandit stronghold removes a raiding threat from nearby trade routes.

**Necromancer's Sanctum:**
- Garrison is undead with living necromancer officers. Undead have no morale but necromancers are fragile.
- Alert mechanic: dead enemies may be re-raised if a necromancer reaches the bodies. Clearing enemies near a necromancer without killing the necromancer first is wasteful.
- Safe rooms must be warded (consumes a resource) or undead wander in during rest.
- Attrition includes a "corruption" resource -- fighting undead accumulates spiritual taint that debuffs the squad over time. Warding items cleanse corruption but are consumable.
- Overworld effect: Clearing the sanctum stops undead incursions in the region.

**Orc War-Camp:**
- Garrison is organized and aggressive. Large patrols, heavy guard posts, lots of reserves. Officers provide significant combat buffs.
- Alert escalation is fast (war horns carry across the entire floor) but orcs don't use traps or subtle tactics.
- Safe rooms are structural weak points the orcs haven't fortified (collapsed tunnels, narrow crevices too small for orc patrols).
- Attrition is raw combat damage. Orc encounters hit hard. Equipment durability drops fast.
- Overworld effect: Clearing the war-camp removes an occupying army from territory.

**Cultist Temple:**
- Garrison is fanatical. Enemies don't rout or surrender. Officers are priests who buff and heal.
- Alert mechanic includes a **ritual timer** -- at high alert, cultists begin a summoning ritual. If the player doesn't reach the ritual chamber in time, a powerful entity is summoned that roams the floor.
- Safe rooms are desecrated shrines that the player must cleanse (one-time consumable cost) before they're safe to rest in.
- Attrition includes morale pressure. Cultist enemies have fear effects. Extended exposure to the temple environment itself drains morale.
- Overworld effect: Clearing the temple disrupts the cult's power base and may prevent a larger-scale summoning event.

### 7.2 Dungeon Outcomes and Overworld Consequences

- **Full clear:** The faction loses their stronghold. Overworld territory controlled by the faction shrinks. The cleared dungeon may become a player-controlled outpost (tie-in with Idea 6, Basecamp Descent).
- **Partial clear (retreat):** The faction is weakened but retains the stronghold. If the player returns later, dead enemies are still dead (the garrison doesn't fully rebuild), but surviving enemies have reorganized and alert is reset. Repeated expeditions can chip away at a dungeon the player can't clear in one run.
- **Failed run (total squad loss):** The faction is emboldened. Overworld aggression from that faction increases. The dungeon's garrison may be slightly reinforced (limited -- it's still a finite garrison, but reserves from the overworld trickle in).

### 7.3 Preparation and Overworld Resources

Success in a garrison-attrition dungeon depends heavily on overworld preparation:

- **Consumable stockpile:** Bringing more potions, repair kits, and ward items extends operational range. Overworld economy directly feeds dungeon capability.
- **Squad roster depth:** Having multiple trained squads available for rotation is more valuable than one maxed-out squad. Overworld squad management (training, equipping, diversifying) directly impacts dungeon strategy.
- **Intelligence gathering:** Overworld scouting, captured prisoners, or purchased maps can provide starting intelligence about a dungeon's layout, faction composition, and garrison strength before the player enters. Pre-run information reduces in-dungeon scouting costs.

---

## 8. Design Risks and Mitigations

| Risk | Mitigation |
|---|---|
| **Attrition feels punishing, not strategic** | Generous early-floor recovery items. Safe rooms are common on Floor 1 to teach the mechanic without pressure. Retreat is always available and preserves progress. |
| **Optimal play is too cautious (rest after every fight)** | Rest has escalating garrison response costs. Patrol shifts during rest create new obstacles. Players who over-rest face a more organized garrison. |
| **Garrison complexity overloads the player** | Alert tier is always visible. Room types are visually distinct. Patrol routes are predictable once observed. The player doesn't need to understand the full system to play -- they just need to know "fighting raises alert, alert makes things harder." |
| **Runs are too long** | Floor count is adjustable per dungeon. Quick-clear bonuses reward efficiency. Partial runs have value (loot, permanent garrison reduction). |
| **Squad rotation feels mandatory** | Single-squad runs are viable but harder -- the attrition curve is steeper. Recovery items and safe rooms exist to support single-squad play. Rotation is an advantage, not a requirement. |
| **Alert system is too opaque** | Visual/audio cues for alert changes (distant horns, increased torch activity, barricades appearing). UI indicator showing current alert tier. Enemy behavior shifts are visible (patrol routes changing, reserves moving through corridors). |

---

## 9. Implementation Priority

A suggested build order, from minimum viable to full feature set:

1. **Multi-floor dungeon with finite enemies** -- Enemies don't respawn. Killing them reduces the dungeon's total population. This is the foundation.
2. **Basic attrition** -- HP and fatigue carry between fights. Safe rooms provide partial recovery. Consumables are finite.
3. **Alert system (basic)** -- Global alert level with 2-3 tiers. Behavior changes are simple (enemies move faster, detection radius increases).
4. **Room types** -- Rooms have identities (barracks, armory, etc.) with appropriate enemy placement and loot.
5. **Squad rotation** -- Multiple squads in the dungeon. Swap at safe rooms.
6. **Alert system (full)** -- All 4 tiers. Runner mechanic. Cross-floor alert propagation. Officer-based coordination.
7. **Faction flavor** -- Different garrison types for different factions with unique mechanics.
8. **Overworld integration** -- Dungeon outcomes affect overworld state. Preparation systems feed into dungeon capability.
