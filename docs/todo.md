__________________________________________
### Update to Gamepay ###

We need to expand on gameplay.

1) Currently, once the game launches, the two main modes are exploration mode and combat mode.
2) I want to expand this. In Exploration mode, the player travels through the world. There are random units placed on the map. They can engage in combat with them.


For now, let's just start with the basics. The map as it currently is works fine for the overworld. What we need is the map with a few units placed on it. Once we walk into the unit, combat starts. That's when the squads are placed. For this part, we still want to use the current squad placement mode in gameplayfactions.go

What I need is:

1) An overworld section. Just the player avatr moves like it currently does. The same GUI as we see in exploration mode
2) Once we walk into a unit, combat mode starts. And then the combatmode gui comes into play. That's when the units get placed. For now, we use the placement from gameplafactions.go

# # # **IMPORTANT** # # # 

- Remove a lot of the commands and GUI elements related to the commands. Those are not needed. After this, review all of the command

- Remove Unused GUI elements and GUI elements you ddont want


### Combat - Important ###

- Handle a unit dying correctly. Currently, it still stays in the squad


________________________________________________
# Testing 

**Combat Testing**

- Confirm that Targeting Works Correctly in Combat. Such as a unit correctly attacking rows
- Ensure that ranged attacks work correctly. Only units in range should attack. If it does work, make sure squads
can't attack twice. Once with ranged units and once with melee


# Cleanup

- tilebatch.go and tilerenderer.go use magic numbers in NewTileBatch


- ecsutil.go seems to have redundancies/duplicate functionality. Look at that. 
- Get Rid of common.Monsters. All the other unit related tags shoudl take care of it. 
- Remove UserMessage component. Not needed
- Remove Quality. DrawableShape Quality should be callled something different
- func CursorPosition(playerPos coords.LogicalPosition) (int, int) in graphictypes requires a change. Leftover from how throwables are handled. 
- CombatController has a lot of artifacts of the old roguelike prior to the change. Specifically in regards to throwables. Find a way tohandle that. 
- Input Package has a lot of leftovers from the old roguelike
- Sprite, Tile, and Rendering Batches allocates default sizes for the slices. Look at those. Determine how many we need. Have a larger default size if needed (i.e, NewSpriteBatch)
- combatqueries.go still has functions which search the entire ecs space



# Bug Fixes
- Units of the same faction should not be able to occupy the same square
- Throwables are completely broken. There is an out of bounds error


# Review



- Identify where there is a possiblity of cache invalidation errors. We started to cache things used for ECS query, so we need to 
- Look at how Tile, sprite, and rendering batching truly works
  * Look at rendering package, 
- Review global position system
- The Inventory is a leftover prior to shifting the game to a turn based squad tactics game from a regular roguelike. Think of what you want to do with the inventory
- Look at all of the const. See which ones should be defined in a file





# Additions 

- Add proper squad/unit cleanup on death
- Add Abilities
- Better Combat Log




# Redundancies?





- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead
- Check on ENABLE_COMBAT_LOG and ENABLE_COMBAT_LOG_EXPORT. Probabably some duplication we can get rid of