__________________________________________
#
# # # **IMPORTANT** # # # 

- Check if we have to mark the caches as dirty, or whether the way the ecs library implements views handles that automatically

- When an encounter starts, we hide the encounter sprites for the combat sprites. Check to make sure that combat can't accidentally start if a combat squad moves into the position of a hidden encounter sprite 

- Handle Entity Cleanup

- Remove a lot of the commands and GUI elements related to the commands. Those are not needed. After this, review all of the command

- Remove Unused GUI elements and GUI elements you ddont want

- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer


### Combat - Important ###

- Handle a unit dying correctly. Currently, it still stays in the squad


________________________________________________
# Testing 



# Cleanup

- tilebatch.go and tilerenderer.go use magic numbers in NewTileBatch



- Get Rid of common.Monsters. All the other unit related tags shoudl take care of it. 

- func CursorPosition(playerPos coords.LogicalPosition) (int, int) in graphictypes requires a change. Leftover from how throwables are handled. 
- CombatController has a lot of artifacts of the old roguelike prior to the change. Specifically in regards to throwables. Find a way tohandle that. 
- Input Package has a lot of leftovers from the old roguelike
- Sprite, Tile, and Rendering Batches allocates default sizes for the slices. Look at those. Determine how many we need. Have a larger default size if needed (i.e, NewSpriteBatch)
- combatqueries.go still has functions which search the entire ecs space



# Bug Fixes
- Throwables are completely broken. There is an out of bounds error


# Review



- Identify where there is a possiblity of cache invalidation errors. We started to cache things used for ECS query, so we need to 



- The Inventory is a leftover prior to shifting the game to a turn based squad tactics game from a regular roguelike. Think of what you want to do with the inventory

- Look at all of the const. See which ones should be defined in a file



- Determine whether we need to mark caches that use ECS views as dirty, or if the ECS library handles it
- behavor and ai package




# Additions 

- Add proper squad/unit cleanup on death
- Add Abilities
- Better Combat Log




# Redundancies?





- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead
- Check on ENABLE_COMBAT_LOG and ENABLE_COMBAT_LOG_EXPORT. Probabably some duplication we can get rid of